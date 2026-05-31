package domain

type Subscription struct {
	ID       string
	Phone    string
	Endpoint string
	Keys     PushKeys
}

type PushKeys struct {
	P256DH string
	Auth   string
}
