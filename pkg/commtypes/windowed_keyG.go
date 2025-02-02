package commtypes

type WindowedKeyG[K any] struct {
	Key    K
	Window Window
}
