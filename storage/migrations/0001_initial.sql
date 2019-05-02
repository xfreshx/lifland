-- +migrate Up
create table players (
	id text primary key,
	points int default 0,
	backers json default null
);

create table tournaments (
	id text primary key,
	deposit int default 0,
	players json default null
);