# Register Personal Finances App


Integrated with Plaid, gets transactions, compares them to the Register tab in the spreadsheet and adds rows as 
needed using the transaction ID as a unique identifier. 

backend MySQL DB that has a few tables including a merchants table that contains a chosen merchants name, a substring to search on, and which category is transaction is added to or subtracted from. 

The configuration file specifies incomes (W2, Social Security, and Pensions). The Budget tab has all categories listed vertically with biweekly and month incomes at the top. I can then make allocations to categories which subtract from the income. When an income source transaction is found, it references the Budget tab and allocates the monies to the specified categories. The Register looks and feels just like a checkbook register using two lines per entry. 