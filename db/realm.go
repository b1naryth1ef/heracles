package db

const REALM_SCHEMA = `
CREATE TABLE IF NOT EXISTS realms (
	id INTEGER PRIMARY KEY,
	name TEXT,
	domain TEXT
);
`

type Realm struct {
	Id   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

func CreateRealm(name string) (*Realm, error) {
	result, err := db.Exec(`INSERT INTO realms (name) VALUES (?);`, name)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Realm{
		Id:   id,
		Name: name,
	}, nil
}

func GetRealmById(id int64) (*Realm, error) {
	var realm Realm
	err := db.Get(&realm, `SELECT * FROM realms WHERE id=?`, id)
	if err != nil {
		return nil, err
	}

	return &realm, nil
}

func GetRealms() ([]Realm, error) {
	var realms []Realm
	err := db.Select(&realms, `SELECT * FROM realms`)
	if realms == nil {
		return make([]Realm, 0), err
	}
	return realms, err
}
