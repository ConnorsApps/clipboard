// Optional: Create an index on the token field for faster lookups
db.tokens.createIndex({ "token": 1 }, { unique: true })

// Optional: Create a TTL index to auto-expire tokens after 30 days
db.tokens.createIndex({ "created_at": 1 }, { expireAfterSeconds: 2592000 })
