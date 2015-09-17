package models

import (
	"database/sql"
	"fmt"
)

var DefaultTemplate = Template{
	ID:       "default",
	Name:     "The Default Template",
	Subject:  "{{.Subject}}",
	Text:     "{{.Text}}",
	HTML:     "{{.HTML}}",
	Metadata: "{}",
}

type TemplatesRepository struct {
	generateGUID guidGeneratorFunc
}

func NewTemplatesRepository(guidGenerator guidGeneratorFunc) TemplatesRepository {
	return TemplatesRepository{
		generateGUID: guidGenerator,
	}
}

func (r TemplatesRepository) Insert(conn ConnectionInterface, template Template) (Template, error) {
	if template.ID != "default" {
		guid, err := r.generateGUID()
		if err != nil {
			panic(err)
		}

		template.ID = guid.String()
	}

	present, err := r.templateWithNameAndClientIDIsPresent(conn, template.Name, template.ClientID)
	if err != nil {
		return template, err
	}
	if present {
		return template, DuplicateRecordError{fmt.Errorf("Template with name %q already exists", template.Name)}
	}

	err = conn.Insert(&template)
	if err != nil {
		return template, err
	}

	return template, nil
}

func (r TemplatesRepository) Get(conn ConnectionInterface, templateID string) (Template, error) {
	template := Template{}
	err := conn.SelectOne(&template, "SELECT * FROM `v2_templates` WHERE `id` = ?", templateID)
	if err != nil {
		if err == sql.ErrNoRows {
			if templateID == DefaultTemplate.ID {
				return DefaultTemplate, nil
			}

			err = RecordNotFoundError{fmt.Errorf("Template with id %q could not be found", templateID)}
		}

		return template, err
	}
	return template, nil
}

func (r TemplatesRepository) Delete(conn ConnectionInterface, templateID string) error {
	template, err := r.Get(conn, templateID)
	if err != nil {
		return err
	}

	_, err = conn.Delete(&template)
	if err != nil {
		return err
	}

	return nil
}

func (r TemplatesRepository) Update(conn ConnectionInterface, template Template) (Template, error) {
	_, err := conn.Update(&template)
	if err != nil {
		return Template{}, err
	}

	return template, nil
}

func (r TemplatesRepository) templateWithNameAndClientIDIsPresent(conn ConnectionInterface, name, clientID string) (bool, error) {
	err := conn.SelectOne(&Template{}, "SELECT * FROM `v2_templates` WHERE `name` = ? AND `client_id` = ?", name, clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r TemplatesRepository) List(conn ConnectionInterface, clientID string) ([]Template, error) {
	templates := []Template{}

	_, err := conn.Select(&templates, "SELECT * FROM `v2_templates` WHERE `client_id` = ?", clientID)

	return templates, err
}
