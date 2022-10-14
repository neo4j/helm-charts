package model

import (
	"gopkg.in/ini.v1"
	"strings"
)

const neo4jConfJvmAdditionalKey = "server.jvm.additional"

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

func (c *Neo4jConfiguration) Update(other Neo4jConfiguration, appendJvmArgs bool) Neo4jConfiguration {
	var jvmArgs = c.jvmArgs
	if len(other.jvmArgs) > 0 {
		if appendJvmArgs {
			jvmArgs = append(jvmArgs, other.jvmArgs...)
		} else {
			jvmArgs = other.jvmArgs
		}
	}

	for k, v := range other.conf {
		c.conf[k] = v
	}
	c.jvmArgs = jvmArgs

	return Neo4jConfiguration{
		jvmArgs: c.jvmArgs,
		conf:    c.conf,
	}
}

func (c *Neo4jConfiguration) UpdateFromMap(other map[string]string, appendJvmArgs bool) Neo4jConfiguration {
	var jvmArgs = c.jvmArgs
	if otherArgsString, found := other["jvmArgs"]; found {
		otherJvmArgs := []string{}
		for _, arg := range strings.Split(otherArgsString, "\n") {
			otherJvmArgs = append(otherJvmArgs, strings.TrimSpace(arg))
		}
		if appendJvmArgs {
			jvmArgs = append(jvmArgs, otherJvmArgs...)
		} else {
			jvmArgs = otherJvmArgs
		}
		delete(other, "jvmArgs")
	}
	for k, v := range other {
		c.conf[k] = v
	}
	c.jvmArgs = jvmArgs

	return Neo4jConfiguration{
		jvmArgs: c.jvmArgs,
		conf:    c.conf,
	}
}
