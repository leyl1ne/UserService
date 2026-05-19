ALTER TABLE IF NOT EXISTS users  
ADD CONSTRAINT fk_company  
FOREIGN KEY (company_id)  
REFERENCES companies(id);