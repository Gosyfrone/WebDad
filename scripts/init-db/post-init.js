// Init : post-service MongoDB
// Exécuté automatiquement au premier démarrage du conteneur

db = db.getSiblingDB(process.env.MONGO_INITDB_DATABASE || 'webdad_post');

// Collection posts
db.createCollection('posts', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['author_id', 'content', 'created_at'],
      properties: {
        author_id: {
          bsonType: 'string',
          description: 'UUID de l\'auteur'
        },
        content: {
          bsonType: 'string',
          maxLength: 280,
          description: 'Contenu du post (280 caractères max)'
        },
        is_hidden: {
          bsonType: 'bool',
          description: 'Masqué par un modérateur'
        },
        hidden_by: {
          bsonType: ['string', 'null'],
          description: 'UUID du modérateur qui a masqué'
        },
        hidden_at: {
          bsonType: ['date', 'null']
        },
        likes_count: {
          bsonType: 'int',
          minimum: 0
        },
        comments_count: {
          bsonType: 'int',
          minimum: 0
        },
        reports_count: {
          bsonType: 'int',
          minimum: 0
        },
        created_at: {
          bsonType: 'date'
        },
        updated_at: {
          bsonType: 'date'
        }
      }
    }
  }
});

// Collection comments
db.createCollection('comments', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['post_id', 'author_id', 'content', 'created_at'],
      properties: {
        post_id: { bsonType: 'string' },
        author_id: { bsonType: 'string' },
        content: {
          bsonType: 'string',
          maxLength: 280
        },
        is_hidden: { bsonType: 'bool' },
        created_at: { bsonType: 'date' },
        updated_at: { bsonType: 'date' }
      }
    }
  }
});

// Collection likes
db.createCollection('likes', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['post_id', 'user_id', 'created_at'],
      properties: {
        post_id: { bsonType: 'string' },
        user_id: { bsonType: 'string' },
        created_at: { bsonType: 'date' }
      }
    }
  }
});

// Collection reports (signalements)
db.createCollection('reports', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['post_id', 'reporter_id', 'reason', 'created_at'],
      properties: {
        post_id: { bsonType: 'string' },
        reporter_id: { bsonType: 'string' },
        reason: {
          bsonType: 'string',
          enum: ['spam', 'harassment', 'misinformation', 'other']
        },
        status: {
          bsonType: 'string',
          enum: ['pending', 'reviewed', 'dismissed']
        },
        created_at: { bsonType: 'date' }
      }
    }
  }
});

// Index
db.posts.createIndex({ author_id: 1 });
db.posts.createIndex({ created_at: -1 });   // tri fil d'actu
db.posts.createIndex({ is_hidden: 1 });

db.comments.createIndex({ post_id: 1 });
db.comments.createIndex({ author_id: 1 });

db.likes.createIndex({ post_id: 1, user_id: 1 }, { unique: true });
db.likes.createIndex({ user_id: 1 });

db.reports.createIndex({ post_id: 1 });
db.reports.createIndex({ status: 1 });

print('post-service: base initialisée');
