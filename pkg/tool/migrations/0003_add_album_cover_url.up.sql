SET search_path TO music, public;

ALTER TABLE covers ADD url text DEFAULT '' NOT NULL;
