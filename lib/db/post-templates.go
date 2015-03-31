package db

import "github.com/draaglom/GleepostAPI/lib/gp"

//CreateTemplate saves this post template to the db, as part of template-set group, returning its id.
func (db *DB) CreateTemplate(group gp.TemplateGroupID, template string) (id gp.TemplateID, err error) {
	s, err := db.prepare("INSERT INTO post_templates (`set`, template) VALUES (?, ?)")
	if err != nil {
		return
	}
	res, err := s.Exec(group, template)
	if err != nil {
		return
	}
	_id, err := res.LastInsertId()
	if err != nil {
		return
	}
	id = gp.TemplateID(_id)
	return
}

//GetTemplate returns a specific template.
func (db *DB) GetTemplate(id gp.TemplateID) (template string, err error) {
	s, err := db.prepare("SELECT template FROM post_templates WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&template)
	return
}

//GetTemplateSet returns all the post templates in this set.
func (db *DB) GetTemplateSet(set gp.TemplateGroupID) (templates []string, err error) {
	s, err := db.prepare("SELECT template FROM post_templates WHERE `set` = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(set)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var tmpl string
		err = rows.Scan(&tmpl)
		if err != nil {
			return
		}
		templates = append(templates, tmpl)
	}
	return
}

//UpdateTemplate saves a new Template
func (db *DB) UpdateTemplate(id gp.TemplateID, group gp.TemplateGroupID, template string) (err error) {
	s, err := db.prepare("REPLACE INTO post_templates (id, `set`, template) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(id, group, template)
	return
}
