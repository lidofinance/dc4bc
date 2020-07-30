package mocks

//go:generate mockgen -source=./../client/state.go -destination=./clientMocks/state_mock.go -package=clientMocks
//go:generate mockgen -source=./../storage/types.go -destination=./storageMocks/storage_mock.go -package=storageMocks
