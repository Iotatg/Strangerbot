package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// stateDoc is the singleton document holding every non-user collection.
type stateDoc struct {
	Waiting  []Waiting         `bson:"waiting"`
	Pairs    map[string]int64  `bson:"pairs"`
	Reports  []Report          `bson:"reports"`
	Feedback []Feedback        `bson:"feedback"`
	Meta     map[string]string `bson:"meta"`
}

// mongoBackend persists users one document each and the rest of the bot state
// in a single document.
type mongoBackend struct {
	client *mongo.Client
	users  *mongo.Collection
	state  *mongo.Collection
}

const stateDocID = "main"

func openMongo(uri, dbName string) (*mongoBackend, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	db := client.Database(dbName)
	return &mongoBackend{
		client: client,
		users:  db.Collection("users"),
		state:  db.Collection("state"),
	}, nil
}

// load reads every user plus the shared state document.
func (m *mongoBackend) load() (data, error) {
	d := newData()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cur, err := m.users.Find(ctx, bson.M{})
	if err != nil {
		return d, err
	}
	for cur.Next(ctx) {
		var u User
		if err := cur.Decode(&u); err != nil {
			cur.Close(ctx)
			return d, err
		}
		d.Users[key(u.ID)] = &u
	}
	if err := cur.Err(); err != nil {
		cur.Close(ctx)
		return d, err
	}
	cur.Close(ctx)

	var sd stateDoc
	err = m.state.FindOne(ctx, bson.M{"_id": stateDocID}).Decode(&sd)
	if err != nil && err != mongo.ErrNoDocuments {
		return d, err
	}
	d.Waiting = sd.Waiting
	d.Pairs = sd.Pairs
	d.Reports = sd.Reports
	d.Feedback = sd.Feedback
	d.Meta = sd.Meta
	return d, nil
}

// saveUser upserts a single user document.
func (m *mongoBackend) saveUser(u *User) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.users.ReplaceOne(ctx, bson.M{"_id": u.ID}, u, options.Replace().SetUpsert(true))
}

// saveState upserts the shared state document.
func (m *mongoBackend) saveState(d *data) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sd := stateDoc{
		Waiting:  d.Waiting,
		Pairs:    d.Pairs,
		Reports:  d.Reports,
		Feedback: d.Feedback,
		Meta:     d.Meta,
	}
	_, _ = m.state.ReplaceOne(ctx, bson.M{"_id": stateDocID}, sd, options.Replace().SetUpsert(true))
}

// deleteUser removes one user document.
func (m *mongoBackend) deleteUser(id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.users.DeleteOne(ctx, bson.M{"_id": id})
}

func (m *mongoBackend) close() {
	_ = m.client.Disconnect(context.Background())
}

// mongoDriverName normalises the configured driver name.
func mongoDriverName(driver string) bool {
	switch driver {
	case "mongo", "mongodb":
		return true
	}
	return false
}
