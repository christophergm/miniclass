-- Create the test database after init-roles.sql has provisioned its owner.
-- This runs during postgres container initialization.

create database miniclass_test owner miniclass_migrator;
alter database miniclass_test owner to miniclass_migrator;

\connect miniclass_test

revoke create on schema public from public;
grant usage on schema public to miniclass_app;
