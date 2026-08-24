-- The default database is created before these scripts run. Make the
-- migrator its owner so local migrations can enforce schema ownership while
-- the API still connects with the non-DDL app role.
select format('alter database %I owner to miniclass_migrator', current_database()) \gexec
