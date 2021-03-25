# lendo-polling-service

This service receives banking Applications via RabbitMQ subscriber, polls their status with a bank and in case the application was processed pushes the application on an outbound RabbitMQ channel. 
