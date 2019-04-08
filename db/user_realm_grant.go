package db

const USER_REALM_GRANT_SCHEMA = `
CREATE TABLE IF NOT EXISTS user_realm_grants (
	user_id INTEGER,
	realm_id INTEGER,
	alias TEXT,

	PRIMARY KEY (user_id, realm_id)
);
`

type UserRealmGrant struct {
	UserId  int64   `json:"user_id" db:"user_id"`
	RealmId int64   `json:"realm_id" db:"realm_id"`
	Alias   *string `json:"alias" db:"alias"`
}

func CreateUserRealmGrant(userId int64, realmId int64, alias *string) (*UserRealmGrant, error) {
	_, err := db.Exec(`
		INSERT INTO user_realm_grants (user_id, realm_id, alias)
		VALUES (?, ?, ?);
	`, userId, realmId, alias)
	if err != nil {
		return nil, err
	}

	return &UserRealmGrant{
		UserId:  userId,
		RealmId: realmId,
		Alias:   alias,
	}, nil
}

func GetUserRealmGrantByRealmName(userId int64, realmName string) (*UserRealmGrant, error) {
	var grant UserRealmGrant

	err := db.Get(&grant, `
		SELECT urg.* FROM user_realm_grants urg
		JOIN realms r ON r.id = urg.realm_id
		WHERE r.name = ? AND urg.user_id = ?
	`, realmName, userId)
	if err != nil {
		return nil, err
	}

	return &grant, nil
}
