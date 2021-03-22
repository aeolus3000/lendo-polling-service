module github.com/aeolus3000/lendo-polling-service

go 1.16

require (
	github.com/aeolus3000/lendo-sdk v1.0.2
	github.com/golang/protobuf v1.5.0
	github.com/google/uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	google.golang.org/protobuf v1.26.0
)

replace (
	github.com/aeolus3000/lendo-sdk v1.0.2 => /home/simon/projects/lendo-sdk
)