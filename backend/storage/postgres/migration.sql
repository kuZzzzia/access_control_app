create database access;

create table files
(
    id            uuid                                               not null
        constraint files_pkey
            primary key,
    created_at    timestamp with time zone default CURRENT_TIMESTAMP not null,
    deleted_at    timestamp with time zone,
    name          varchar(256)                                       not null,
    user_id       uuid                                               not null,
    bucket_name   varchar(64)                                        not null,
    extension     varchar(16)                                        not null,
    size          integer                                            not null,
    people_number integer                                            not null
);