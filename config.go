package gom

import "strings"

type config struct {
	FieldMap     map[string]string `yaml:"nameAlias"`
	Abbreviation map[string]string `yaml:"abbreviation"`
	Querys       sqlList           `yaml:"query"`
	Execs        sqlList           `yaml:"exec"`
}

func (c *config) init() error {
	err := c.Querys.init(c)
	if err != nil {
		return err
	}

	err = c.Execs.init(c)
	if err != nil {
		return err
	}
	return nil
}

var sqlStarts = []string{"select", "update", "delete", "insert", "replace",
	"create", "drop", "alter"}

func isSQL(sql string) bool {
	lower := strings.ToLower(sql)
	for _, s := range sqlStarts {
		if strings.HasPrefix(lower, s) {
			return true
		}
	}
	return false
}

func (c *config) findQuery(name string) *varQuery {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	q := c.Querys.find(name)
	if q != nil {
		return q
	}

	if isSQL(name) {
		q = &varQuery{SQL: name, Name: "dynamic generate varQuery"}
		if err := q.init(c); err != nil {
			return nil
		}
		return q
	}

	return nil
}

func (c *config) findExec(name string) *varQuery {
	return c.Execs.find(name)
}
