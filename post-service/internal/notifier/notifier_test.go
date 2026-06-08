package notifier

import (
	"reflect"
	"testing"
)

func TestParseMentions(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"aucune mention", "bonjour tout le monde", nil},
		{"une mention", "salut @bob !", []string{"bob"}},
		{"plusieurs mentions", "@alice et @bob_42 venez voir", []string{"alice", "bob_42"}},
		{"déduplication insensible à la casse", "@Bob @bob @BOB", []string{"Bob"}},
		{"email non capté comme mention", "écris à jean@exemple.com", nil},
		{"handle trop court ignoré", "@ab @bob", []string{"bob"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseMentions(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseMentions(%q) = %v ; attendu %v", tc.content, got, tc.want)
			}
		})
	}
}

// TestNoop : le no-op n'émet rien et ne panique pas (post-service autonome).
func TestNoop(t *testing.T) {
	var n Notifier = Noop{}
	n.Emit(Event{Type: TypeLike}) // ne doit rien faire
}
