# Auction Service

Este projeto implementa um serviço de leilão com backend em Go e MongoDB. Ele utiliza Docker Compose para configurar e rodar a aplicação e o banco de dados MongoDB.

## Pré-requisitos

Certifique-se de ter o Docker e Docker Compose instalados em sua máquina. Você pode seguir as instruções de instalação aqui:
- [Instalar Docker](https://docs.docker.com/get-docker/)
- [Instalar Docker Compose](https://docs.docker.com/compose/install/)

## Configuração

O projeto usa um arquivo `.env` para configurar variáveis de ambiente. O arquivo `.env` deve estar localizado em `cmd/auction/.env`.

Exemplo de conteúdo do arquivo `.env`:

```dotenv
BATCH_INSERT_INTERVAL=20s
MAX_BATCH_SIZE=4
AUCTION_INTERVAL=20s
AUCTION_DURATION=30

MONGO_INITDB_ROOT_USERNAME=admin
MONGO_INITDB_ROOT_PASSWORD=admin
MONGODB_URL=mongodb://admin:admin@mongodb:27017/auctions?authSource=admin
MONGODB_DB=auctions
```

## Instruções para rodar o projeto
1. Clone o repositório:
```sh
git clone git@github.com:obrunogonzaga/labs-auction-goexpert.git
cd labs-auction-goexpert
```

2. Certifique-se de que o arquivo .env esteja no local correto:
```sh
cp exemplo.env cmd/auction/.env
```

3. Construa e inicie os serviços com Docker Compose:
```sh
docker-compose up --build
```

4. Acesse a aplicação:
A aplicação estará rodando na porta 8080 do seu localhost. Você pode acessá-la em http://localhost:8080.

5. Para parar os serviços:
Pressione CTRL+C no terminal onde os serviços estão rodando ou execute:
```sh
docker-compose down
```

## Estrutuca do projeto
- cmd/auction/: Contém o código principal da aplicação e o arquivo .env.
- Dockerfile: Define a imagem Docker para a aplicação.
- docker-compose.yml: Define os serviços Docker e suas configurações.

## Testes

Para rodar os testes, certifique-se de que o serviço MongoDB esteja rodando. Em seguida, você pode executar os testes utilizando o comando:
```sh
go test ./...
```

Certifique-se de que suas variáveis de ambiente de teste estejam configuradas corretamente no arquivo .env.