package gom

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

func (c *config) findQuery(name string) *varQuery {
	return c.Querys.find(name)
}

func (c *config) findExec(name string) *varQuery {
	return c.Execs.find(name)
}
