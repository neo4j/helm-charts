#!/bin/bash
set -e

#echo "export PATH=./tools:.:$PATH" >> $BASH_ENV
mkdir -p $BUILD_ARTIFACTS
mkdir -p tools

echo $GCLOUD_SERVICE_KEY | base64 -d > $SERVICE_KEY_FILE
gcloud auth activate-service-account $GCLOUD_SERVICE_ACCOUNT --key-file=$SERVICE_KEY_FILE
gcloud auth configure-docker --quiet gcr.io

az login --service-principal --username "$SP_ID" --password "$SP_PASSWORD" --tenant "$TENANT_ID"

helm lint

echo "GKE SETUP"
export INSTANCE_NAME=$INSTANCE-$BUILD_NUMBER
./tools/test/provision-k8s.sh $INSTANCE_NAME

NAMESPACE=ns-$BUILD_NUMBER
kubectl create ns $NAMESPACE

kubectl create secret generic neo4j-service-key \
--namespace $NAMESPACE \
--from-file=credentials=$SERVICE_KEY_FILE

# This secret is injected in the test process to demonstrate that
# config works.  This is just any valid config we can check inside of
# a running system.
kubectl create secret generic my-secret-config \
--namespace $NAMESPACE \
--from-literal=NEO4J_dbms_transaction_concurrent_maximum=0

echo "export ACCOUNT_NAME=$ACCOUNT_NAME" >> azure-credentials.sh
echo "export ACCOUNT_KEY=$ACCOUNT_KEY" >> azure-credentials.sh
kubectl create secret generic azure-credentials \
--namespace $NAMESPACE \
--from-file=credentials=azure-credentials.sh

helm package .
chart_archive=$(ls neo4j*.tgz)
cp *.tgz $BUILD_ARTIFACTS/

echo "Installing $chart_archive (STANDALONE SCENARIO)"
helm install $NAME_STANDALONE -f deployment-scenarios/ci/standalone.yaml $chart_archive \
 --namespace $NAMESPACE \
 -v 3 | tee -a $BUILD_ARTIFACTS/INSTALL-standalone.txt

echo "Installing $chart_archive (AZURE RESTORE SCENARIO)"
helm install $NAME_RESTORE -f deployment-scenarios/ci/single-instance-restore.yaml $chart_archive \
 --namespace $NAMESPACE \
 -v 3 | tee -a $BUILD_ARTIFACTS/INSTALL-restore.txt

kubectl rollout status --namespace $NAMESPACE StatefulSet/$NAME_STANDALONE-neo4j-core --watch | tee -a $BUILD_ARTIFACTS/wait-standalone.txt

helm test $NAME_STANDALONE --namespace $NAMESPACE --logs | tee -a $BUILD_ARTIFACTS/TEST-STANDALONE.txt

helm package tools/backup
chart_archive=$(ls neo4j*.tgz)
cp *.tgz $BUILD_ARTIFACTS/

helm install standalone-backup-gcp tools/backup \
--namespace $NAMESPACE \
--set neo4jaddr=$NAME_STANDALONE-neo4j.$NAMESPACE.svc.cluster.local:6362 \
--set bucket=$BUCKET/$BUILD_NUMBER/ \
--set database="neo4j\,system" \
--set cloudProvider=gcp \
--set jobSchedule="0 */12 * * *" \
--set secretName=neo4j-service-key

sleep 5
kubectl get all -n $NAMESPACE
echo "Taking a backup"
kubectl create job --namespace $NAMESPACE --from=cronjob/standalone-backup-gcp-job gcp-hot-backup

helm install standalone-backup-azure tools/backup \
--namespace $NAMESPACE \
--set neo4jaddr=$NAME_STANDALONE-neo4j.$NAMESPACE.svc.cluster.local:6362 \
--set bucket=$AZURE_CONTAINER_NAME/build-$BUILD_NUMBER/ \
--set database="neo4j\,system" \
--set cloudProvider=azure \
--set jobSchedule="0 */12 * * *" \
--set secretName=azure-credentials

sleep 5
kubectl get all -n $NAMESPACE
echo "Taking a backup"
kubectl create job --namespace $NAMESPACE --from=cronjob/standalone-backup-azure-job azure-hot-backup

# If "latest" backup pointer files exist in a dir that is specific to this
# build number, we should be good.
sleep 65
kubectl get all -n $NAMESPACE
export LOGFILE=$BUILD_ARTIFACTS/gcp-backup.log
kubectl get job --namespace $NAMESPACE | tee -a $LOGFILE
helm status standalone-backup-gcp --namespace $NAMESPACE | tee -a $LOGFILE

backup_pods=$(kubectl get pods --namespace $NAMESPACE | grep gcp-hot-backup | sed 's/ .*$//' | head -n 1)

echo "Backup pods $backup_pods" | tee -a $LOGFILE
kubectl describe pod --namespace $NAMESPACE "$backup_pods" | tee -a $LOGFILE
kubectl logs --namespace $NAMESPACE "$backup_pods" | tee -a $LOGFILE

gsutil ls "$BUCKET/$BUILD_NUMBER/neo4j/neo4j-latest.tar.gz" 2>&1 | tee -a $BUILD_ARTIFACTS/gcp-backup.log
gsutil ls "$BUCKET/$BUILD_NUMBER/system/system-latest.tar.gz" 2>&1 | tee -a $BUILD_ARTIFACTS/gcp-backup.log

#If "latest" backup pointer files exist in a dir that is specific to this
#build number, we should be good.
kubectl get all -n $NAMESPACE
export LOGFILE=$BUILD_ARTIFACTS/azure-backup.log
kubectl get job --namespace $NAMESPACE | tee -a $LOGFILE
helm status standalone-backup-azure --namespace $NAMESPACE | tee -a $LOGFILE

backup_pods=$(kubectl get pods --namespace $NAMESPACE | grep azure-hot-backup | sed 's/ .*$//' | head -n 1)

echo "Backup pods $backup_pods" | tee -a $LOGFILE
kubectl describe pod --namespace $NAMESPACE "$backup_pods" | tee -a $LOGFILE
kubectl logs --namespace $NAMESPACE "$backup_pods" | tee -a $LOGFILE

az storage blob list -c $AZURE_CONTAINER_NAME --account-name neo4jpublic --prefix build-$BUILD_NUMBER/ | tee -a files.txt
cat files.txt >> $LOGFILE
total_files=$(cat files.txt | grep name | wc -l)
if [ $total_files = 4 ] ; then
echo "Test pass" ;
else
echo "$total_files total files on Azure storage; failed"
exit 1
fi

export NEO4J_PASSWORD=mySecretPassword
kubectl logs --namespace $NAMESPACE $NAME_RESTORE-neo4j-core-0 \
-c restore-from-backup | tee -a $BUILD_ARTIFACTS/restore.log

# Wait for instance to come alive.
kubectl rollout status --namespace $NAMESPACE StatefulSet/$NAME_RESTORE-neo4j-core --watch | tee -a $BUILD_ARTIFACTS/wait-standalone-restore.txt

echo "MATCH (n) RETURN count(n) as n;" | kubectl run -i --rm cypher-shell \
--namespace $NAMESPACE \
--image=neo4j:4.2.0-enterprise --restart=Never \
--command -- ./bin/cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
-a neo4j://$NAME_RESTORE-neo4j.$NAMESPACE.svc.cluster.local 2>&1 | tee restore-result.log
cp restore-result.log $BUILD_ARTIFACTS/

# Strip all cypher shell output down to a single integer result cound for n
export record_count=$(cat restore-result.log | egrep '^[0-9]+$')
echo "record_count=$record_count"
if [ "$record_count" -gt "1000" ] ; then
echo "Test pass" ;
else
echo "Test FAIL with record count $record_count"
exit 1
fi
