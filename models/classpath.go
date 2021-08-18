package models

import (
	"strings"
	"time"

	"github.com/toolkits/pkg/logger"
	"github.com/toolkits/pkg/str"
)

type Classpath struct {
	Id       int64  `json:"id"`
	Path     string `json:"path"`
	Note     string `json:"note"`
	Preset   int    `json:"preset"`
	CreateAt int64  `json:"create_at"`
	CreateBy string `json:"create_by"`
	UpdateAt int64  `json:"update_at"`
	UpdateBy string `json:"update_by"`
}

type ClasspathNode struct {
	Id       int64            `json:"id"`
	Path     string           `json:"path"`
	Note     string           `json:"note"`
	Preset   int              `json:"preset"`
	Children []*ClasspathNode `json:"children"`
}

func (c *Classpath) TableName() string {
	return "classpath"
}

func (c *Classpath) Validate() error {
	if str.Dangerous(c.Path) {
		return _e("Classpath path has invalid characters")
	}

	if strings.Contains(c.Path, " ") {
		return _e("Classpath path has invalid characters")
	}

	if str.Dangerous(c.Note) {
		return _e("Classpath note has invalid characters")
	}

	return nil
}

func (c *Classpath) Add() error {
	if err := c.Validate(); err != nil {
		return err
	}

	num, err := ClasspathCount("path=?", c.Path)
	if err != nil {
		return err
	}

	if num > 0 {
		return _e("Classpath %s already exists", c.Path)
	}

	now := time.Now().Unix()
	c.CreateAt = now
	c.UpdateAt = now
	return DBInsertOne(c)
}

func ClasspathCount(where string, args ...interface{}) (num int64, err error) {
	num, err = DB.Where(where, args...).Count(new(Classpath))
	if err != nil {
		logger.Errorf("mysql.error: count classpath fail: %v", err)
		return num, internalServerError
	}
	return num, nil
}

func (c *Classpath) Update(cols ...string) error {
	if err := c.Validate(); err != nil {
		return err
	}

	_, err := DB.Where("id=?", c.Id).Cols(cols...).Update(c)
	if err != nil {
		logger.Errorf("mysql.error: update classpath(id=%d) fail: %v", c.Id, err)
		return internalServerError
	}

	return nil
}

func ClasspathTotal(query string) (num int64, err error) {
	if query != "" {
		q := "%" + query + "%"
		num, err = DB.Where("path like ?", q).Count(new(Classpath))
	} else {
		num, err = DB.Count(new(Classpath))
	}

	if err != nil {
		logger.Errorf("mysql.error: count classpath fail: %v", err)
		return 0, internalServerError
	}

	return num, nil
}

func ClasspathGets(query string, limit, offset int) ([]Classpath, error) {
	session := DB.Limit(limit, offset).OrderBy("path")
	if query != "" {
		q := "%" + query + "%"
		session = session.Where("path like ?", q)
	}
	var objs []Classpath
	err := session.Find(&objs)
	if err != nil {
		logger.Errorf("mysql.error: query classpath fail: %v", err)
		return objs, internalServerError
	}

	if len(objs) == 0 {
		return []Classpath{}, nil
	}

	return objs, nil
}

func ClasspathGetAll() ([]Classpath, error) {
	var objs []Classpath
	err := DB.Find(&objs)
	if err != nil {
		logger.Errorf("mysql.error: query classpath fail: %v", err)
		return objs, internalServerError
	}

	if len(objs) == 0 {
		return []Classpath{}, nil
	}

	return objs, nil
}

func ClasspathGet(where string, args ...interface{}) (*Classpath, error) {
	var obj Classpath
	has, err := DB.Where(where, args...).Get(&obj)
	if err != nil {
		logger.Errorf("mysql.error: query classpath(%s)%+v fail: %s", where, args, err)
		return nil, internalServerError
	}

	if !has {
		return nil, nil
	}

	return &obj, nil
}

func ClasspathGetsByPrefix(prefix string) ([]Classpath, error) {
	var objs []Classpath
	err := DB.Where("path like ?", prefix+"%").OrderBy("path").Find(&objs)
	if err != nil {
		logger.Errorf("mysql.error: query classpath fail: %v", err)
		return objs, internalServerError
	}

	if len(objs) == 0 {
		return []Classpath{}, nil
	}

	return objs, nil
}

// Del classpath的删除，前提是挂载的机器、配置的采集策略都要提前删除
func (c *Classpath) Del() error {
	num, err := ClasspathResourceCount("classpath_id=?", c.Id)
	if err != nil {
		return err
	}

	if num > 0 {
		return _e("There are still resources under the classpath")
	}

	num, err = CollectRuleCount("classpath_id=?", c.Id)
	if err != nil {
		return err
	}

	if num > 0 {
		return _e("There are still collect rules under the classpath")
	}

	session := DB.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return err
	}

	if _, err := session.Exec("DELETE FROM classpath_favorite WHERE classpath_id=?", c.Id); err != nil {
		logger.Errorf("mysql.error: delete classpath_favorite fail: %v", err)
		return err
	}

	if _, err := session.Exec("DELETE FROM classpath WHERE id=?", c.Id); err != nil {
		logger.Errorf("mysql.error: delete classpath fail: %v", err)
		return err
	}

	return session.Commit()
}

func (c *Classpath) AddResources(idents []string) error {
	count := len(idents)
	for i := 0; i < count; i++ {
		err := ClasspathResourceAdd(c.Id, strings.TrimSpace(idents[i]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Classpath) DelResources(idents []string) error {
	return ClasspathResourceDel(c.Id, idents)
}

func ClasspathNodeGets(query string) ([]*ClasspathNode, error) {
	session := DB.OrderBy("path")
	if query != "" {
		q := "%" + query + "%"
		session = session.Where("path like ?", q)
	}
	var objs []Classpath
	err := session.Find(&objs)
	if err != nil {
		logger.Errorf("mysql.error: query classpath fail: %v", err)
		return []*ClasspathNode{}, internalServerError
	}

	if len(objs) == 0 {
		return []*ClasspathNode{}, nil
	}
	pcs := ClasspathNodeAllChildren(objs)

	return pcs, nil
}

func (cp *Classpath) DirectChildren() ([]Classpath, error) {
	var pcs []Classpath
	objs, err := ClasspathGetsByPrefix(cp.Path)
	if err != nil {
		logger.Errorf("mysql.error: query prefix classpath fail: %v", err)
		return []Classpath{}, internalServerError
	}
	if len(objs) < 2 {
		return []Classpath{}, nil
	}

	pre := objs[1]
	path := pre.Path[len(objs[0].Path):]
	pre.Path = path
	pcs = append(pcs, pre)

	for _, cp := range objs[2:] {
		has := strings.HasPrefix(cp.Path, pre.Path)
		if !has {
			path := cp.Path[len(objs[0].Path):]
			pre.Path = path
			pcs = append(pcs, pre)
			pre = cp
		}
	}

	return pcs, nil
}

func ClasspathNodeAllChildren(cps []Classpath) []*ClasspathNode {
	var node ClasspathNode
	for _, cp := range cps {
		ListInsert(cp, &node)
	}

	return node.Children
}

func ListInsert(obj Classpath, node *ClasspathNode) {
	path := obj.Path
	has := true
	for {
		if len(node.Children) == 0 {
			break
		}
		children := node.Children[len(node.Children)-1]
		prefix := children.Path
		has = strings.HasPrefix(path, prefix)
		if !has {
			break
		}
		path = path[len(prefix):]
		node = children
	}

	newNode := ToClasspathNode(obj, path)
	node.Children = append(node.Children, &newNode)
}

func ToClasspathNode(cp Classpath, path string) ClasspathNode {
	var obj ClasspathNode
	obj.Id = cp.Id
	obj.Path = path
	obj.Note = cp.Note
	obj.Preset = cp.Preset
	obj.Children = []*ClasspathNode{}

	return obj
}
