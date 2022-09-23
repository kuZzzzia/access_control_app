Запуск posgres

docker run -d -e POSTGRES_PASSWORD=root -e POSTGRES_DATABASE=access -e PGADMIN_LISTEN_PORT=6000 -p 6000:5432 --name access postgres

Запуск minio

docker run -p 9000:9000 -d -p 9001:9001 -e "MINIO_ROOT_USER=minio99" -e "MINIO_ROOT_PASSWORD=minio123" quay.io/minio/minio server /data --console-address ":9001"
