package serialization_test

import (
	"hidden-prompt-backend/pkg/serialization"
	"testing"
	"time"
)

func Test_Serialize(t *testing.T) {
	type User struct {
		Name      string    `text:"name"`
		Age       int       `text:"age,omitempty"`
		Pass      string    `text:"pass,omitempty"`
		Timestamp time.Time `text:"time"`
	}

	u := struct {
		Users    []User `text:"users"`
		NewUsers []User `text:"new_users"`
	}{
		Users: []User{
			{Name: "Alice", Age: 10, Timestamp: time.Now()},
			{Name: "Alan", Timestamp: time.Now()},
		},
		NewUsers: []User{
			{Name: "Right", Age: 29, Timestamp: time.Now()},
		},
	}

	s, _ := serialization.Serialize(u)

	t.Logf("Resp:\n%s", s)
}
