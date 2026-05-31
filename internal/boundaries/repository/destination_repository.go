package repository

type DestinationRepository interface {
	GetAll() ([]string, error)
}
