local http = require('http')
local json = require('json')
local inspect = require('inspect')

local function neo4j_discovery(applet)
    local req = http.request.parse(applet)

    local req_host = req:get_header("host")
    if not req_host then
        core.Alert("request doesn't have host header!?")
        return
    end
    for k, v in req:get_headers() do
        core.Debug(k .. ": " .. v)
    end
    local req_host_port = ""
    local req_port = req:get_header("x-forwarded-port")
    core.Debug("original reqhost: "..req_host)
    -- because the js driver will provide a default port if we don't!
    if req_host:find(":", 1, true) ~= nil then
        req_host_and_port = req_host
        local host_match, _ = string.gmatch(req_host, '([^:]+)', 2)
        req_host = host_match()
        req_port = host_match()
    elseif req:get_header("x-scheme") == "https" and req:get_header("x-forwarded-port") == "443" then
        req_host_and_port = req_host .. ":443"
    elseif req:get_header("x-scheme") == "http" and req:get_header("x-forwarded-port") == "80" then
        req_host_and_port = req_host .. ":80"
    end
    core.Info(string.format("translating discovery request with req_host_and_port : %s", req_host_and_port))

    -- Use http_backend from HAProxy config to resolve IP as DNS is unavailable to Lua script
    local http_backend = core.backends["neo4j-http"]
    if not http_backend then
        core.Alert("cannot find backend named 'neo4j-http'")
        return
    end

    -- get the first server in our backend
    local server = nil
    for k, v in pairs(http_backend.servers) do
        server = v
        break
    end
    local host = server:get_addr()
    if not host then
        core.Alert(string.format("can't get a host value for server %s", server))
        return
    end
    local res, err = http.get { url = string.format("http://%s", host),
                                headers = { host = host,
                                            ["accept"] = { "application/json" },
                                            ["connection"] = { "close" } }
    }
    if res then
        for k, v in res:get_headers() do
            core.Debug(k .. ": " .. v)
        end
    else
        core.Debug(err)
    end
    local response_body = json.decode(res.content)
    core.Info("Transforming Neo4j discovery response:")
    core.Info(inspect(response_body))
    local neo4j_scheme = "neo4j://"
    local bolt_scheme = "bolt://"
    local http_scheme = "http://"
    if req:get_header("x-forwarded-port") == "443" then
        neo4j_scheme = "neo4j+s://"
        bolt_scheme = "bolt+s://"
        http_scheme = "https://"
    end
    for index, value in pairs(response_body) do
        if(index == "bolt_routing") then
            response_body[index] = neo4j_scheme..req_host_and_port
        elseif(index == "dbms/cluster") then
            response_body[index] = http_scheme..req_host_and_port.."/dbms/cluster"
        elseif(index == "transaction") then
            response_body[index] = http_scheme..req_host_and_port.."/db/{databaseName}/tx"
        elseif(index == "db/cluster") then
            response_body[index] = http_scheme..req_host_and_port.."/db/{databaseName}/cluster"
        elseif(index == "bolt_direct") then
            response_body[index] = bolt_scheme..req_host_and_port
        end

    end
    core.Info("Transformed Neo4j discovery response:")
    core.Info(inspect(response_body))
    http.response.create {
        status_code = 200,
        content = json.encode(response_body)}:send(applet)

end
core.register_service('neo4j_discovery', 'http', neo4j_discovery)