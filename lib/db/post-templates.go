package db

import "github.com/draaglom/GleepostAPI/lib/gp"

//CreateTemplate saves this post template to the db, as part of template-set group, returning its id.
func (db *DB) CreateTemplate(group gp.TemplateGroupID, template string) (id gp.TemplateID, err error) {
	s, err := db.prepare("INSERT INTO post_templates (set, template) VALUES (?, ?)")
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
