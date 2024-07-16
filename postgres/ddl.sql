CREATE TABLE locations (
  serial_number VARCHAR (10) PRIMARY KEY, 
  latitude decimal NOT NULL,
  longitude decimal NOT NULL,
  timestamp TIMESTAMP NOT NULL
);
