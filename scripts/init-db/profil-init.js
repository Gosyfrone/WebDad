// Init : profil-service MongoDB
// Exécuté automatiquement au premier démarrage du conteneur

db = db.getSiblingDB(process.env.MONGO_INITDB_DATABASE || 'webdad_profil');

db.createCollection('profiles', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['user_id', 'username'],
      properties: {
        user_id:         { bsonType: 'string' },
        username:        { bsonType: 'string' },
        bio:             { bsonType: 'string', maxLength: 160 },
        avatar_url:      { bsonType: 'string' },
        website:         { bsonType: 'string' },
        location:        { bsonType: 'string' },
        followers_count: { bsonType: 'int', minimum: 0 },
        following_count: { bsonType: 'int', minimum: 0 },
        posts_count:     { bsonType: 'int', minimum: 0 },
        created_at:      { bsonType: 'date' },
        updated_at:      { bsonType: 'date' }
      }
    }
  }
});

db.profiles.createIndex({ user_id: 1 }, { unique: true });
db.profiles.createIndex({ username: 1 }, { unique: true });

db.profiles.insertOne({
  user_id:         '00000000-0000-0000-0000-000000000001',
  username:        'admin',
  bio:             'Administrateur de WebDad',
  avatar_url:      '',
  website:         '',
  location:        '',
  followers_count: 0,
  following_count: 0,
  posts_count:     0,
  created_at:      new Date(),
  updated_at:      new Date()
});

print('profil-service: base initialisée');