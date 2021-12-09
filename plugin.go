package bedprox

type Plugin interface {
	Load() error
	Unload() error

	OnPlayerJoin() error
	OnPlayerLeave() error
}
