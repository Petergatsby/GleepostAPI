package lib

import "testing"

func TestMessageNormalize(t *testing.T) {
	type normalizetest struct {
		before string
		after  string
	}
	tests := []normalizetest{
		{
			before: "",
			after:  "",
		},
		{
			before: "hey",
			after:  "hey",
		},
		{
			before: "hey <@user:123|@patrick>",
			after:  "hey @patrick",
		},
		{
			before: "hey <>",
			after:  "hey <>",
		},
		{
			before: "hey <|>",
			after:  "hey <|>",
		},
		{
			before: "hey <patrick>",
			after:  "hey <patrick>",
		},
		{
			before: "hey <patrick> I mean <@patrick>",
			after:  "hey <patrick> I mean <@patrick>",
		},
		{
			before: "hey <patrick|> I mean <@patrick|>",
			after:  "hey <patrick|> I mean <@patrick|>",
		},
		{
			before: "hey <@123|@patrick> I mean <@patrick|>",
			after:  "hey @patrick I mean <@patrick|>",
		},
		{
			before: "hey <@user:123|@patrick> and <@user:456|@tade>",
			after:  "hey @patrick and @tade",
		},
	}
	for _, test := range tests {
		if normalizeMessage(test.before) != test.after {
			t.Fatalf("Expected {%s} to transform into {%s} but actually got {%s}\n", test.before, test.after, normalizeMessage(test.before))
		}
	}
}
