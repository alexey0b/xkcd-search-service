FROM postgres:15-alpine

ENV POSTGRES_DB=test_db
ENV POSTGRES_USER=user  
ENV POSTGRES_PASSWORD=password

COPY init.sql /docker-entrypoint-initdb.d/

EXPOSE 5432