package mocks

//go:generate mockgen -source=./../client/modules/state/state.go -destination=./clientMocks/state_mock.go -package=clientMocks
//go:generate mockgen -source=./../client/modules/keystore/keystore.go -destination=./clientMocks/keystore_mock.go -package=clientMocks
//go:generate mockgen -source=./../storage/types.go -destination=./storageMocks/storage_mock.go -package=storageMocks
//go:generate mockgen -source=./../qr/qr.go -destination=./qrMocks/qr_mock.go -package=qrMocks
//go:generate mockgen -source=./../client/services/fsmservice/fsmservice.go -destination=./serviceMocks/fsmservice_mock.go -package=serviceMocks
//go:generate mockgen -source=./../client/repositories/operation/operation.go -destination=./repoMocks/operation_mock.go -package=repoMocks
//go:generate mockgen -source=./../client/repositories/signature/signature.go -destination=./repoMocks/signature_mock.go -package=repoMocks
//go:generate mockgen -source=./../client/services/operation/operation.go -destination=./serviceMocks/operation_mock.go -package=serviceMocks
//go:generate mockgen -source=./../client/services/signature/signature.go -destination=./serviceMocks/signature_mock.go -package=serviceMocks
