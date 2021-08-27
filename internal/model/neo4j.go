package model

import "gopkg.in/ini.v1"

const neo4jConfJvmAdditionalKey = "dbms.jvm.additional"

type Neo4jConfiguration struct {
	conf    map[string]string
	jvmArgs []string
}

func (c *Neo4jConfiguration) JvmArgs() []string {
	return c.jvmArgs
}

func (c *Neo4jConfiguration) Conf() map[string]string {
	return c.conf
}

func (c *Neo4jConfiguration) PopulateFromFile(filename string) (*Neo4jConfiguration, error) {
	yamlFile, err := ini.ShadowLoad(filename)
	if err != nil {
		return nil, err
	}
	defaultSection := yamlFile.Section("")

	jvmAdditional, err := defaultSection.GetKey(neo4jConfJvmAdditionalKey)
	if err != nil {
		return nil, err
	}
	c.jvmArgs = jvmAdditional.StringsWithShadows("\n")
	c.conf = defaultSection.KeysHash()
	delete(c.conf, neo4jConfJvmAdditionalKey)

	return c, err
}

func (c *Neo4jConfiguration) Update(other Neo4jConfiguration) Neo4jConfiguration {
	var jvmArgs []string
	if len(other.jvmArgs) > 0 {
		jvmArgs = other.jvmArgs
	} else {
		jvmArgs = c.jvmArgs
	}
	for k, v := range other.conf {
		c.conf[k] = v
	}

	return Neo4jConfiguration{
		jvmArgs: jvmArgs,
		conf:    c.conf,
	}
}
