package models

// Publisher represents a book publisher with basic contact info.
type Publisher struct {
	ID      int64  `json:"id"      db:"id"`
	Name    string `json:"name"    db:"name"`
	Country string `json:"country" db:"country"`

	Meta
}

func (Publisher) TableName() string { return "publishers" }
func (Publisher) IDField() string   { return "id" }

func (Publisher) FilterFieldMap() map[string]string {
	return map[string]string{
		"id":      "publishers.id",
		"name":    "publishers.name",
		"country": "publishers.country",
		"state":   "publishers.state",
	}
}

type PublisherArgs struct {
	RequestCommons
}

type PublisherBody struct {
	Publisher *Publisher `json:"publisher"`
	Included  Included   `json:"included"`
}

type PublishersBody struct {
	Publishers []*Publisher `json:"publishers"`
	Included   Included     `json:"included"`
	Pagination Pagination   `json:"pagination"`
}
