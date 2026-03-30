package entity

type Group struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}
