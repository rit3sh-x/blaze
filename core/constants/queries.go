package constants

import "fmt"

const TEST_QUERY = "SELECT 1"

const DROP_PUBLIC_SCHEMA = `
DO
$func$
BEGIN
EXECUTE 'DROP SCHEMA public CASCADE';
EXECUTE 'CREATE SCHEMA public';
END
$func$;
`

const BLAZE_TABLE_QUERY = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS "_blaze_migrations" (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
checksum VARCHAR(64) UNIQUE NOT NULL,
applied_at TIMESTAMP(3) UNIQUE NOT NULL DEFAULT now(),
migration_name TEXT UNIQUE NOT NULL,
step_count INT NOT NULL DEFAULT 1
);
`

const FETCH_ALL_MIGRATIONS = `
SELECT
id,
checksum,
applied_at,
migration_name,
step_count
FROM "_blaze_migrations"
ORDER BY applied_at DESC;
`

const ALL_ENUMS_QUERY = `
SELECT 
t.typname AS enum_name,
e.enumlabel AS enum_value,
e.enumsortorder AS sort_order
FROM pg_type t
JOIN pg_enum e ON t.oid = e.enumtypid
JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE n.nspname = 'public'
ORDER BY enum_name, e.enumsortorder;
`

const FETCH_AVAILABLE_TABLES = `
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY table_schema, table_name;
`

func TableTypes(input string) string {
	return fmt.Sprintf(`
    SELECT 
    c.column_name,
    c.udt_name AS base_type,
    c.is_nullable,
    c.column_default,
    c.ordinal_position
    FROM information_schema.columns c
    LEFT JOIN pg_type t
    ON c.udt_name = t.typname
    LEFT JOIN pg_enum e
    ON t.oid = e.enumtypid
    WHERE c.table_name = "%s"
    AND c.table_schema NOT IN ('pg_catalog', 'information_schema')
    GROUP BY c.column_name, c.data_type, c.is_nullable, c.column_default, t.typname, c.ordinal_position, c.udt_name
    ORDER BY c.ordinal_position;
    `, input)
}

func TableConstraints(input string) string {
	return fmt.Sprintf(`
    SELECT 
    tc.constraint_type,
    kcu.column_name,
    tc.constraint_name
    FROM information_schema.table_constraints tc
    JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
    WHERE tc.table_name = "%s"
    AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')
    ORDER BY tc.constraint_type, kcu.column_name;
    `, input)
}

func TableRelations(input string) string {
	return fmt.Sprintf(`
    SELECT
    kcu.column_name AS fk_column,
    ccu.table_name AS referenced_table,
    ccu.column_name AS referenced_column,
    rc.update_rule,
    rc.delete_rule,
    tc.constraint_name
    FROM information_schema.table_constraints AS tc
    JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
    JOIN information_schema.referential_constraints AS rc
    ON tc.constraint_name = rc.constraint_name
    AND tc.table_schema = rc.constraint_schema
    JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
    WHERE tc.constraint_type = 'FOREIGN KEY'
    AND tc.table_name = "%s";
    `, input)
}

func TableIndexes(input string) string {
	return fmt.Sprintf(`
    SELECT 
    i.relname AS index_name,
    idx.indisunique AS is_unique,
    idx.indisprimary AS is_primary,
    array_to_string(array_agg(a.attname), ', ') AS columns
    FROM pg_class t
    JOIN pg_index idx ON t.oid = idx.indrelid
    JOIN pg_class i ON i.oid = idx.indexrelid
    JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(idx.indkey)
    WHERE t.relkind = 'r' 
    AND t.relname = "%s"
    GROUP BY i.relname, idx.indisunique, idx.indisprimary
    ORDER BY i.relname;
    `, input)
}