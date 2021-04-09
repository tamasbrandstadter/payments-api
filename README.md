# Payments API
This application is created for managing financial accounts in payments solutions.

Available operations:
- Opening accounts (creating new accounts for customers)
- Listing accounts
- Get an account by id
- Freeze an account
- Delete an account
- Get balance from account 

The application also handles financial transactions in async and concurrent manner for:
- Depositing amount to account
- Withdrawing amount from account
- Transferring amount between accounts

## Technology stack
- Programming language: `go 1.16`
- Message broker: `RabbitMQ`
- Database: `PostgreSQL`
- Cache: `Redis`
- Containerisation: `Docker`
- Container orchestrator: `Kubernetes`, `minikube`
- Integration test and local development setup: `docker-compose`

## Workflow
Customers, accounts and transactions are stored in the database. REST API is created and publicly exposed for CRUD operations with
stored records.

The message broker is responsible for routing incoming deposit, withdraw and transfer messages, which will be consumed by the
application from their dedicated queues. The balance will be updated in the database and in the cache. 
Database transactions are enabled on different isolation levels, in case of an error the transaction will be rolled back and (if applicable) retried.

If a transaction is successful an audit record will be saved to the database, and an event will be sent to the corresponding
topic for notifying the customer also asynchronously.

## Architectural diagram
![Alt text](./diagram.svg)

## Running locally
* The project uses `go`, therefore you must install it first on your local machine if you want to develop the
  application. Also install `Docker` if you don't have it locally.

* If you want to run the application locally simply execute `make up` in the project root folder. 
  Docker-compose will start the database, message broker, cache containers.

* If you want to run the application from your IDEA then use port 8080 and set these environment variables:
  - `DB_USER=myuser`
  - `DB_NAME=mydb`
  - `DB_PASSWORD=mypassword`
  - `MQ_USER=guest`
  - `MQ_PASSWORD=guest`
  - `MQ_HOST=localhost`
  - `CACHE_HOST=localhost`
  - `CACHE_PASSWORD=securepass`
  - `DB_HOST=localhost`

* You can reach the API via the following endpoints:
  - GET `/accounts/{id}` - get an account
  - GET `/accounts` - get stored accounts
  - GET `/accounts/{id}/balance` - get balance from an account from the cache or database
  - POST `/accounts` - create new account
  - PUT `/accounts/{id}/freeze` - freeze an account
  - DELETE `/accounts/{id}` - delete account

* You can check the published messages on management console via `http://localhost:15672/`.

* You can reach the `database on port 5432`. `Cache` is reachable on port `6379`.

* If you want to build a Docker image use `make tag` (and optionally `make push`).

## Deployment
* This application and the underlying infrastructure deployed to a Kubernetes cluster. Install `minikube` if you don't have it locally.

* Kubernetes descriptor YAML files can be found in `deploy` and `kubernetes` folders.

* If you want to deploy, then follow these steps:
  - `minikube start --memory=6g —cpus=2` (start k8s cluster)
  - `make add-host` (DNS entry for ingress)
  - `make kube-infra-up` (this will apply infra descriptors)
  - `make kube-api-up` (this will apply API descriptors)
  - `minikube service payments-api --url -n payments` (public IP assigning)

* Note this issue if you encounter problems in volume mounting for database container starts: [minikube#4634](https://github.com/kubernetes/minikube/issues/4634)

* Optionally use `minikube tunnel` if you want to check the management console for the message broker.

### Testing
* Unit and integration tests are implemented as part of the project.

* To run them execute `make test`. This will create the test Docker containers and run the integration tests against them.