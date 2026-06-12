/**
 * Registre i18n de Breezy (mécanisme maison léger, sans dépendance).
 *
 * Deux briques :
 *   - LOCALES : la liste des langues disponibles (source de vérité du sélecteur).
 *   - messages : le dictionnaire `clé → texte` par locale.
 *
 * Pour AJOUTER UNE LANGUE (es, it, zh…) :
 *   1. ajouter une entrée dans LOCALES ;
 *   2. ajouter le bloc correspondant dans `messages` (mêmes clés que `fr`).
 * Aucun autre changement : le provider, le hook `useT` et le sélecteur
 * s'adaptent automatiquement (cf. language-provider.tsx / language-selector.tsx).
 *
 * Calqué sur lib/themes.ts (registre piloté par un provider client + localStorage),
 * la dimension MODE étant gérée par next-themes et la dimension LANGUE par
 * LanguageProvider.
 */

/** Une langue sélectionnable. */
export interface LocaleDef {
  /** Code court (ISO 639-1), posé sur <html lang> et persisté. */
  id: string
  /** Libellé natif affiché dans le sélecteur. */
  label: string
  /** Drapeau (emoji) pour la pastille du sélecteur. */
  flag: string
}

/**
 * Langues disponibles. La PREMIÈRE est la langue par défaut (rendu serveur).
 * Pour l'instant : Français + English. Les suivantes (Español, Italiano, 中文…)
 * s'ajoutent ici + dans `messages`.
 */
export const LOCALES: LocaleDef[] = [
  { id: 'fr', label: 'Français', flag: '🇫🇷' },
  { id: 'en', label: 'English', flag: '🇬🇧' },
]

export type Locale = (typeof LOCALES)[number]['id']

/** Langue par défaut = première du registre. Utilisée au rendu serveur. */
export const DEFAULT_LOCALE: Locale = LOCALES[0].id

/** Clé localStorage de persistance de la préférence de langue. */
export const LOCALE_STORAGE_KEY = 'breezy-locale'

/** Garde de type : la valeur est-elle une locale connue ? */
export function isLocale(value: unknown): value is Locale {
  return typeof value === 'string' && LOCALES.some((l) => l.id === value)
}

/**
 * Dictionnaire `clé → texte` par locale.
 *
 * Convention de clés : `namespace.key` (ex. `nav.feed`, `auth.login.title`).
 * `fr` est la référence (toutes les clés y existent) ; les autres locales en
 * sont une traduction. Une clé manquante dans une locale retombe sur `fr`, puis
 * sur la clé brute (cf. translate()).
 */
export type Messages = Record<string, string>

export const messages: Record<Locale, Messages> = {
  fr: {
    // — Commun —
    'common.user': 'Utilisateur',
    'common.username_fallback': 'utilisateur',
    'common.not_connected': 'non connecté',
    'common.logout': 'Se déconnecter',
    'common.cancel': 'Annuler',
    'common.save': 'Enregistrer',
    'common.loading': 'Chargement…',
    'common.retry': 'Réessayer',
    'common.close': 'Fermer',
    'media.download': "Télécharger l'image",
    'media.previous': 'Image précédente',
    'media.next': 'Image suivante',
    'media.speed': 'Vitesse de lecture',
    'media.speed_normal': 'Normal',

    // — Navigation —
    'nav.feed': 'Fil',
    'nav.explore': 'Explorer',
    'nav.notifications': 'Notifications',
    'nav.messages': 'Messages',
    'nav.profil': 'Profil',
    'nav.moderation': 'Modération',
    'nav.admin': 'Administration',
    'nav.settings': 'Paramètres',
    'nav.home': 'Accueil',
    'nav.bookmarks': 'Signets',
    'nav.open_menu': 'Ouvrir le menu de navigation',
    'nav.post': 'Breezer',
    'nav.compose': 'Composer une publication',
    'nav.search': 'Recherche',

    // — Mode visiteur (non connecté) —
    'visitor.login': 'Se connecter',
    'visitor.register': "S'inscrire",
    'visitor.cta_aria': 'Se connecter ou s’inscrire',
    'visitor.prompt_title': 'Rejoins Breezy',
    'visitor.prompt_desc':
      'Connecte-toi ou crée un compte pour aimer, commenter, reposter et suivre des comptes.',

    'bookmarks.title': 'Signets',
    'bookmarks.all': 'Tous',
    'bookmarks.default_name': 'Mes signets',
    'bookmarks.save': 'Enregistrer',
    'bookmarks.save_to': 'Ranger dans une collection',
    'bookmarks.organize': 'Ranger…',
    'bookmarks.saved_to': 'Enregistré dans « {name} »',
    'bookmarks.removed': 'Retiré des signets',
    'bookmarks.add_aria': 'Enregistrer dans les signets',
    'bookmarks.remove_aria': 'Retirer des signets',
    'bookmarks.new_collection': 'Nouvelle collection',
    'bookmarks.new_collection_placeholder': 'Nom de la collection',
    'bookmarks.create': 'Créer',
    'bookmarks.no_collections': 'Aucune collection pour le moment.',
    'bookmarks.manage': 'Gérer la collection',
    'bookmarks.rename': 'Renommer',
    'bookmarks.delete': 'Supprimer',
    'bookmarks.delete_title': 'Supprimer la collection',
    'bookmarks.delete_confirm': 'Les posts ne seront plus dans cette collection (ils restent publiés).',
    'bookmarks.empty_title': 'Aucun signet',
    'bookmarks.empty_message': 'Enregistrez des posts pour les retrouver ici.',
    'bookmarks.empty_collection': 'Cette collection est vide.',
    'bookmarks.load_error': 'Impossible de charger les signets.',

    // — Rôles —
    'role.user': 'Utilisateur',
    'role.moderator': 'Modérateur',
    'role.administrator': 'Administrateur',

    // — Sélecteur de thème —
    'theme.title': 'Thème',
    'theme.appearance': 'Apparence',
    'theme.toggle_aria': 'Basculer entre le mode clair et sombre',
    'theme.system': 'Mode système',
    'theme.on': 'Activé',
    'theme.off': 'Désactivé',
    // — Thème personnalisé —
    'theme.customize': 'Personnaliser le thème',
    'theme.customize_hint': 'Choisissez les couleurs de votre Breezy.',
    'theme.target_background': "Fond d'écran",
    'theme.target_background_hint': "Couleur de fond de l'application",
    'theme.target_text': 'Couleur du texte',
    'theme.target_text_hint': '« Pour toi », « Abonnements »…',
    'theme.target_primary': "Boutons d'action",
    'theme.target_primary_hint': '« Breezer », « Suivre »…',
    'theme.reset': 'Réinitialiser',
    'theme.back': 'Retour',
    'theme.hex': 'Code hexadécimal',
    'theme.brightness': 'Luminosité',
    'theme.color_wheel_aria': 'Roue de sélection de couleur',

    // — Sélecteur de langue —
    'lang.title': 'Langue',
    'lang.select_aria': 'Choisir la langue',

    // — Paramètres —
    'settings.title': 'Paramètres',
    'settings.account_title': 'Paramètres du compte',
    'settings.account_desc':
      'Personnalisez votre expérience Breezy.',
    'filters.title': 'Mots filtrés',
    'filters.desc':
      "Masquez les posts du fil qui contiennent un mot ou une expression que vous ne voulez pas voir.",
    'filters.placeholder': 'Ex. one piece',
    'filters.input_aria': 'Mot ou expression à filtrer',
    'filters.add': 'Ajouter le filtre',
    'filters.remove_aria': 'Retirer le filtre {word}',
    'filters.empty': 'Aucun mot filtré pour le moment.',

    // — Pages légales —
    'legal.badge': 'Informations légales',
    'legal.updated': 'Dernière mise à jour : {date}',
    'legal.back': 'Retour à Breezy',
    'legal.section': 'Légal',
    'legal.mentions': 'Mentions légales',
    'legal.cgu': 'CGU',
    'legal.confidentialite': 'Confidentialité',

    // — Notifications —
    'notifications.title': 'Notifications',
    'notifications.unread_aria': '{count} notifications non lues',
    'notifications.empty': 'Aucune notification pour le moment.',
    'notifications.load_more': 'Voir plus',
    'notifications.like_one': '{name} a aimé votre publication',
    'notifications.like_other': '{name} et {count} autres personnes ont aimé votre publication',
    'notifications.comment_one': '{name} a commenté votre publication',
    'notifications.comment_other':
      '{name} et {count} autres personnes ont commenté votre publication',
    'notifications.reply_one': '{name} a répondu à votre commentaire',
    'notifications.reply_other':
      '{name} et {count} autres personnes ont répondu à votre commentaire',
    'notifications.repost_one': '{name} a reposté votre publication',
    'notifications.repost_other':
      '{name} et {count} autres personnes ont reposté votre publication',
    'notifications.quote': '{name} a cité votre publication',
    'notifications.mention': '{name} vous a mentionné',
    'notifications.message_mention_one': '{name} vous a mentionné dans un message',
    'notifications.message_mention_other': '{name} vous a mentionné dans {count} messages',
    'notifications.follow_one': "{name} s'est abonné(e) à vous",
    'notifications.follow_other': "{name} et {count} autres personnes se sont abonnées à vous",

    // — Mentions (@handle) —
    'mentions.view_in_search': 'Voir dans la recherche',
    'mentions.user_not_found': 'Compte introuvable',
    'notifications.follow_request': '{name} demande à vous suivre',
    'notifications.follow_request_accepted': '{name} a accepté votre demande',
    'notifications.follow_request_accept_confirm': 'Vous avez accepté la demande de {name}',
    'notifications.follow_request_accept_confirm_prefix': 'Vous avez accepté la demande de',
    'notifications.post_purge_warning':
      'Un de vos tweets retiré par la modération sera supprimé définitivement dans environ un mois.',
    'notifications.accept': 'Accepter',
    'notifications.reject': 'Refuser',

    // — Détail d'une publication —
    'common.back': 'Retour',
    'post.detail_title': 'Publication',
    'post.not_found': 'Publication introuvable.',

    // — Pages stub (états vides) —
    'notifications.heading': 'Rien pour le moment',
    'notifications.desc': 'Vos notifications (likes, abonnements, mentions) apparaîtront ici.',
    'messages.title': 'Messages',
    'messages.heading': 'Aucune conversation',
    'messages.empty_desc': 'Démarrez une discussion avec le bouton +.',
    'messages.new': 'Nouveau',
    'messages.new_dm': 'Nouveau message',
    'messages.new_dm_desc':
      'Recherchez une personne pour démarrer une conversation chiffrée de bout en bout.',
    'messages.new_group': 'Nouveau groupe',
    'messages.new_group_desc':
      'Nommez le groupe et ajoutez des membres. Le nom et les messages sont chiffrés de bout en bout.',
    'messages.new_community': 'Créer une communauté',
    'messages.discover': 'Découvrir des communautés',
    'messages.discover_desc':
      'Rejoignez des communautés publiques. Vous y entrez en lecture seule jusqu’à être promu.',
    'messages.create_group': 'Créer le groupe',
    'messages.create_community': 'Créer',
    'messages.group_name_placeholder': 'Nom du groupe',
    'messages.community_name_placeholder': 'Nom de la communauté',
    'messages.community_note':
      'Le nom est public et les messages sont lisibles par un administrateur (communauté semi-publique).',
    'messages.community_admin_note':
      'Communauté semi-publique : les messages peuvent être lus par un administrateur.',
    'messages.search_user_placeholder': 'Rechercher une personne (@ pour l’identifiant)…',
    'messages.search_community_placeholder': 'Rechercher une communauté…',
    'messages.no_users': 'Aucun utilisateur trouvé.',
    'messages.no_communities': 'Aucune communauté trouvée.',
    'messages.select_title': 'Vos messages',
    'messages.select_desc': 'Sélectionnez une conversation ou démarrez-en une nouvelle.',
    'messages.composer_placeholder': 'Écrivez un message…',
    'messages.send': 'Envoyer',
    'messages.edit': 'Modifier',
    'messages.editing': 'Modification du message',
    'messages.cancel_edit': 'Annuler la modification',
    'messages.save_edit': 'Enregistrer',
    'messages.edited': 'Modifié',
    'messages.hide_original': "Masquer l'original",
    'messages.add_attachment': 'Joindre une image ou une vidéo',
    'messages.open_image': "Agrandir l'image",
    'messages.attachment_failed': 'Pièce jointe indisponible',
    'messages.attachment_preview': 'Pièce jointe',
    'messages.back': 'Retour',
    'messages.info': 'Détails',
    'messages.members': 'Membres',
    'messages.members_count': '{count} membres',
    'messages.invite': 'Inviter',
    'messages.rename': 'Renommer',
    'messages.leave': 'Quitter',
    'messages.delete': 'Supprimer',
    'messages.remove_member': 'Exclure',
    'messages.promote': 'Promouvoir (peut écrire)',
    'messages.demote': 'Rétrograder (lecture seule)',
    'messages.join': 'Rejoindre',
    'messages.joined': 'Rejoint',
    'messages.you': 'vous',
    'messages.message_action': 'Message',
    'messages.read_only': 'Lecture seule — vous ne pouvez pas écrire dans cette communauté.',
    'messages.key_missing':
      'Clé indisponible sur cet appareil : cette conversation a été créée sur un autre appareil.',
    'messages.decrypt_failed': 'Message chiffré (clé indisponible)',
    'messages.passphrase.setup_title': 'Protégez vos messages',
    'messages.passphrase.setup_desc':
      'Définissez une phrase de passe à retenir pour accéder à vos messages privés sur vos autres appareils. Elle chiffre votre clé : nous ne pouvons pas la récupérer si vous l’oubliez.',
    'messages.passphrase.unlock_title': 'Débloquez vos messages',
    'messages.passphrase.unlock_desc':
      'Saisissez la phrase de passe définie sur votre premier appareil pour déchiffrer vos messages privés ici.',
    'messages.passphrase.field': 'Phrase de passe',
    'messages.passphrase.field_placeholder': 'Votre phrase de passe',
    'messages.passphrase.confirm_field': 'Confirmez la phrase de passe',
    'messages.passphrase.show': 'Afficher la phrase de passe',
    'messages.passphrase.hide': 'Masquer la phrase de passe',
    'messages.passphrase.setup_cta': 'Je définis ma phrase de passe',
    'messages.passphrase.unlock_cta': 'Déverrouiller',
    'messages.passphrase.working': 'Veuillez patienter…',
    'messages.passphrase.strength.weak': 'Faible',
    'messages.passphrase.strength.fair': 'Correcte',
    'messages.passphrase.strength.good': 'Bonne',
    'messages.passphrase.strength.strong': 'Excellente',
    'messages.passphrase.too_weak':
      'Phrase trop simple : visez au moins 10 caractères et un mélange de mots/chiffres/symboles.',
    'messages.passphrase.mismatch': 'Les deux phrases de passe ne correspondent pas.',
    'messages.passphrase.wrong': 'Phrase de passe incorrecte.',
    'messages.passphrase.setup_failed': 'Impossible d’enregistrer la phrase de passe. Réessayez.',
    'messages.passphrase.unlock_failed': 'Impossible de déverrouiller. Vérifiez votre phrase de passe.',
    'messages.passphrase.warning': 'À retenir absolument : sans elle, vos messages restent inaccessibles ailleurs.',
    'messages.no_messages_title': 'Aucun message',
    'messages.no_messages_desc': 'Envoyez le premier message 👋',
    'messages.new_messages_divider': 'Nouveaux messages',
    'messages.you_prefix': 'Vous : {text}',
    'messages.mentioned_you': '{name} vous a mentionné',
    'messages.unread_aria': 'Messages non lus',
    'messages.badge_aria': '{count} conversations avec des messages non lus',
    'messages.actions_aria': 'Actions de la conversation',
    'messages.pin': 'Épingler',
    'messages.unpin': 'Désépingler',
    'messages.mute': 'Mettre en sourdine',
    'messages.unmute': 'Réactiver les notifications',
    'messages.muted_aria': 'Conversation en sourdine',
    'messages.delete_for_me': 'Supprimer pour moi',
    'messages.delete_failed': 'Suppression impossible.',
    'messages.search_in_conversation': 'Rechercher dans la conversation…',
    'messages.search_no_results': 'Aucun message trouvé',
    'messages.type_dm': 'Message privé',
    'messages.type_group': 'Groupe',
    'messages.type_community': 'Communauté',
    'messages.role_owner': 'Propriétaire',
    'messages.role_admin': 'Admin',
    'messages.role_member': 'Membre',
    'messages.role_talker': 'Intervenant',
    'messages.role_viewer': 'Spectateur',
    'messages.load_failed': 'Impossible de charger les messages.',
    'messages.send_failed': 'Échec de l’envoi du message.',
    'messages.edit_failed': 'Échec de la modification du message.',
    'messages.dm_failed': 'Impossible de démarrer la conversation.',
    'messages.group_failed': 'Impossible de créer le groupe.',
    'messages.community_failed': 'Impossible de créer la communauté.',
    'messages.join_failed': 'Impossible de rejoindre la communauté.',
    'messages.action_failed': 'Action impossible.',
    'admin.heading': 'Administration Breezy',
    'admin.desc': 'La gestion des utilisateurs et des rôles arrivera ici.',
    'admin.subtitle': 'Gérez les comptes, les rôles et les accès.',
    'admin.search_placeholder': 'Rechercher par e-mail',
    'admin.empty': 'Aucun compte trouvé.',
    'admin.error': 'Impossible de charger les comptes.',
    'admin.status_active': 'Actif',
    'admin.status_banned': 'Banni',
    'admin.action_ban': 'Bannir',
    'admin.action_unban': 'Réactiver',
    'admin.you': 'Vous',
    'admin.role_changed': 'Rôle mis à jour : {role}',
    'admin.banned_toast': 'Compte banni.',
    'admin.unbanned_toast': 'Compte réactivé.',
    'admin.action_failed': 'Action impossible.',
    'admin.access_denied': 'Accès refusé',
    'admin.access_denied_desc': 'Cette page est réservée aux administrateurs.',
    'admin.role_note': "Un changement de rôle prend effet à la prochaine reconnexion de l'utilisateur (rafraîchissement du jeton).",
    'moderation.heading': 'Centre de modération',
    'moderation.desc': 'Les signalements et les actions de modération apparaîtront ici.',
    'moderation.subtitle': 'Gardez l’ordre sur Breezy : tweets retirés et comptes.',
    'moderation.tab_posts': 'Tweets supprimés',
    'moderation.tab_accounts': 'Comptes',
    'moderation.posts_empty': 'Aucun tweet dans la corbeille.',
    'moderation.posts_error': 'Impossible de charger la corbeille.',
    'moderation.restore': 'Restaurer',
    'moderation.purge': 'Supprimer définitivement',
    'moderation.purge_title': 'Supprimer définitivement ce tweet ?',
    'moderation.purge_desc':
      'Cette action est irréversible : le tweet et ses likes, commentaires et reposts seront effacés pour de bon.',
    'moderation.removed_by': 'Retiré par {who} · {when}',
    'moderation.purge_scheduled': 'Purge définitive prévue le {when}',
    'moderation.expiring_soon': 'Bientôt purgé',
    'moderation.restored_toast': 'Tweet restauré.',
    'moderation.purged_toast': 'Tweet supprimé définitivement.',
    'moderation.access_denied': 'Accès refusé',
    'moderation.access_denied_desc': 'Cette page est réservée aux modérateurs et administrateurs.',
    'moderation.ban_restricted': 'Un modérateur ne peut bannir qu’un utilisateur.',
    'moderation.banned_since': 'Banni depuis {when}',
    'moderation.account_delete': 'Supprimer le compte',
    'moderation.account_delete_soon': 'Suppression définitive du compte (RGPD) — à venir.',
    'moderation.account_delete_title': 'Supprimer définitivement ce compte ?',
    'moderation.account_delete_desc':
      'Effacement RGPD irréversible : profil, tweets, messages, médias et relations seront supprimés de tous les services. Tapez « {username} » pour confirmer.',
    'moderation.account_delete_confirm_label': 'Nom d’utilisateur à confirmer',
    'moderation.account_deleted_toast': 'Compte supprimé définitivement.',
    'moderation.account_delete_failed': 'Échec de la suppression (effacement partiel possible).',
    'admin.infra_subtitle': 'Supervision de l’infrastructure et des services.',
    'admin.infra_heading': 'État des services',
    'admin.infra_desc':
      'L’état des conteneurs Docker et l’uptime des services apparaîtront ici prochainement.',
    'admin.monitoring_error': 'Impossible de charger l’état des services.',
    'admin.monitoring_updated': 'Mis à jour {when}',
    'admin.status_up': 'En ligne',
    'admin.status_down': 'Hors ligne',
    'admin.latency': 'Latence',
    'admin.uptime': 'Uptime',
    'admin.uptime_d': 'j',
    'admin.uptime_h': 'h',
    'admin.uptime_m': 'min',
    'admin.uptime_s': 's',

    // — Recherche / Tendances (colonne droite) —
    'search.placeholder': 'Rechercher',
    'trends.title': 'Tendances',
    'trends.trending': 'Tendance',
    'trends.t1.category': 'Technologie',
    'trends.t1.posts': '12,4 K posts',
    'trends.t2.category': 'Dev',
    'trends.t2.posts': '8,1 K posts',
    'trends.t3.category': 'Cloud',
    'trends.t3.posts': '5,6 K posts',

    // — Navigation mobile —
    'nav.main_aria': 'Navigation principale',
    'post.create_aria': 'Créer un post',

    // — 404 —
    'notfound.message': "Cette page n'existe pas.",
    'notfound.back': "Retour à l'accueil",

    // — Authentification (commun) —
    'auth.search_aria': 'Rechercher',
    'auth.email_label': 'Adresse e-mail',
    'auth.email_placeholder': 'toi@exemple.com',
    'auth.password_label': 'Mot de passe',
    'auth.show_password': 'Afficher le mot de passe',
    'auth.hide_password': 'Masquer le mot de passe',
    'auth.err.email_required': "L'adresse e-mail est requise.",
    'auth.err.email_invalid': 'Saisis une adresse e-mail valide.',
    'auth.err.email_max': "L'adresse e-mail est limitée à 50 caractères.",
    'auth.err.identifier_required': "L'adresse e-mail ou le username est requis.",
    'auth.err.password_required': 'Le mot de passe est requis.',
    'auth.err.password_min': 'Le mot de passe doit contenir au moins 8 caractères.',
    'auth.err.network': "Impossible de contacter l'API. Réessaie dans un instant.",

    // — Login —
    'auth.login.demo.kicker': 'Fil en direct',
    'auth.login.demo.heading': 'Retrouve ton monde.',
    'auth.login.demo.post1': 'Nouvelle playlist, nouveaux débats, même énergie Breezy.',
    'auth.login.demo.views': '18.4K vues',
    'auth.login.demo.joined': 'Noa a rejoint la conversation',
    'auth.login.demo.joined_sub': 'Découvre les sujets qui montent ce soir.',
    'auth.login.demo.rank': '#{n} sur Breezy',
    'auth.login.demo.interactions': '17 nouvelles interactions',
    'auth.login.demo.interactions_sub': "Ton fil t'attend, frais et vivant.",
    'auth.login.badge': 'Connexion au réseau',
    'auth.login.title': "Reprends ton fil là où tu l'as laissé.",
    'auth.login.subtitle':
      'Connecte-toi à Breezy, retrouve tes messages, tes posts et les conversations qui bougent.',
    'auth.login.identifier_label': 'Adresse e-mail ou Username',
    'auth.login.identifier_placeholder': 'toi@exemple.com ou username',
    'auth.login.forgot': 'Mot de passe oublié ?',
    'auth.login.submit': 'Se connecter',
    'auth.login.submitting': 'Connexion en cours…',
    'auth.login.or': 'Ou se connecter avec',
    'auth.login.no_account': 'Pas encore de compte ?',
    'auth.login.create_account': 'Créer un compte',
    'auth.login.failed': 'La connexion a échoué. Vérifie tes identifiants.',
    'auth.login.account_disabled':
      'Compte désactivé, veuillez contacter le support de Breezy.',

    // — OAuth —
    'auth.oauth.loading': 'Connexion en cours…',
    'auth.oauth.error': 'La connexion a échoué. Réessaie depuis la page de connexion.',
    'auth.oauth.back_to_login': 'Retour à la connexion',

    // — Register —
    'auth.register.demo.kicker': 'Nouveau sur Breezy',
    'auth.register.demo.heading': 'Crée ton espace.',
    'auth.register.demo.you': 'Toi',
    'auth.register.demo.now': 'maintenant',
    'auth.register.demo.post1':
      'Premier post, première vibe, et déjà toute une communauté à rencontrer.',
    'auth.register.demo.welcome': 'Bienvenue',
    'auth.register.demo.opens': "Breezy t'ouvre le fil",
    'auth.register.demo.opens_sub': 'Choisis ton pseudo et commence à publier.',
    'auth.register.demo.to_join': 'À rejoindre',
    'auth.register.demo.community': 'Communauté #{n}',
    'auth.register.demo.comm1': 'Créateurs',
    'auth.register.demo.comm2': 'Campus CESI',
    'auth.register.demo.comm3': 'Dev Distribué',
    'auth.register.demo.alive': 'Ton compte prend vie',
    'auth.register.demo.alive_sub': 'Profil, posts et conversations en quelques secondes.',
    'auth.register.badge': 'Nouveau profil Breezy',
    'auth.register.title': 'Rejoins Breezy et commence à publier.',
    'auth.register.subtitle':
      "Crée ton compte, choisis ton nom d'utilisateur et entre dans le fil.",
    'auth.register.username_label': "Nom d'utilisateur",
    'auth.register.username_tooltip':
      "Le nom d'utilisateur pourra être changé après la création du compte, puis une fois tous les 14 jours.",
    'auth.register.birthdate_label': 'Date de naissance',
    'auth.register.birthdate_tooltip':
      'La date de naissance ne pourra plus être changée une fois le compte créé.',
    'auth.register.gender_label': 'Genre',
    'auth.register.gender_male': 'Homme',
    'auth.register.gender_female': 'Femme',
    'auth.register.password_confirm_label': 'Confirmation du mot de passe',
    'auth.register.show_password_confirm': 'Afficher la confirmation du mot de passe',
    'auth.register.hide_password_confirm': 'Masquer la confirmation du mot de passe',
    'auth.register.password_help':
      '8 caractères min., majuscule, minuscule, chiffre et caractère spécial.',
    'auth.register.submit': 'Créer mon compte',
    'auth.register.submitting': 'Création du compte…',
    'auth.register.or': 'Ou créer mon compte avec',
    'auth.register.have_account': 'Tu as déjà un compte ?',
    'auth.register.err.username_required': "Le nom d'utilisateur est requis.",
    'auth.register.err.username_format':
      '3 à 24 caractères : lettres, chiffres et tiret bas (_) uniquement.',
    'auth.register.err.username_max': "Le nom d'utilisateur est limité à 24 caractères.",
    'auth.register.err.username_reserved': "Ce nom d'utilisateur n'est pas autorisé.",
    'auth.register.err.username_taken': "Ce nom d'utilisateur est déjà pris.",
    'auth.register.err.username_check':
      "Impossible de vérifier le nom d'utilisateur. Réessaie dans un instant.",
    'auth.register.err.birthdate_required': 'La date de naissance est requise.',
    'auth.register.err.birthdate_invalid': 'Saisis une date de naissance valide.',
    'auth.register.err.birthdate_future': 'La date de naissance ne peut pas être dans le futur.',
    'auth.register.err.age': "Tu dois avoir au moins 13 ans pour t'inscrire.",
    'auth.register.err.gender_required': 'Choisis un genre.',
    'auth.register.err.password_max': 'Le mot de passe est limité à 250 caractères.',
    'auth.register.err.password_format':
      '8 caractères minimum, une majuscule, une minuscule, un chiffre et un caractère spécial.',
    'auth.register.err.confirm_required': 'Confirme ton mot de passe.',
    'auth.register.err.confirm_max': 'La confirmation est limitée à 250 caractères.',
    'auth.register.err.confirm_mismatch': 'Les mots de passe ne correspondent pas.',
    'auth.register.err.failed': "L'inscription a échoué. Vérifie les informations saisies.",

    // — Onboarding (finalisation des comptes OAuth sans profil) —
    'onboarding.title': 'Bienvenue sur Breezy 👋',
    'onboarding.subtitle':
      "Choisis ton nom d'utilisateur et indique ta date de naissance pour finaliser ton compte.",
    'onboarding.username_label': "Nom d'utilisateur",
    'onboarding.username_placeholder': 'ex. felipe',
    'onboarding.username_hint': '3 à 24 caractères : lettres, chiffres et tiret bas (_).',
    'onboarding.username_checking': 'Vérification de la disponibilité…',
    'onboarding.username_available': 'Disponible ✓',
    'onboarding.birthdate_label': 'Date de naissance',
    'onboarding.birthdate_disclaimer':
      "En dessous de 18 ans, les contenus sensibles (NSFW) seront masqués. La date de naissance n'est pas modifiable après validation.",
    'onboarding.submit': 'Valider et continuer',
    'onboarding.submitting': 'Enregistrement…',
    'onboarding.err.generic': 'Une erreur est survenue. Réessaie dans un instant.',

    // — Vérification d'e-mail —
    'auth.check_email.title': 'Vérifie ta boîte mail',
    'auth.check_email.subtitle':
      "Un lien de vérification vient d'être envoyé à {email}. Clique dessus pour activer ton compte.",
    'auth.check_email.subtitle_generic':
      "Un lien de vérification vient d'être envoyé à ton adresse. Clique dessus pour activer ton compte.",
    'auth.check_email.hint':
      "Pense à regarder dans tes spams. Le lien expire dans 24 heures.",
    'auth.check_email.resend': 'Renvoyer le lien de vérification',
    'auth.check_email.back_to_login': 'Retour à la connexion',
    'auth.verify.login_blocked':
      "Ton adresse e-mail n'est pas encore vérifiée. Vérifie ta boîte mail pour activer ton compte.",
    'auth.verify.resend_cta': 'Renvoyer le mail de vérification',
    'auth.verify.resending': 'Envoi en cours…',
    'auth.verify.resend_done':
      "Si un compte non vérifié correspond à cette adresse, un e-mail vient d'être envoyé.",
    'auth.verify.resend_label': 'Ton adresse e-mail',
    'auth.verify.loading_title': 'Vérification en cours…',
    'auth.verify.loading_desc': 'Un instant, on confirme ton adresse e-mail.',
    'auth.verify.success_title': 'Adresse vérifiée !',
    'auth.verify.success_desc':
      'Ton compte est activé et te voilà connecté. Bienvenue sur Breezy !',
    'auth.verify.go_to_app': 'Accéder à Breezy',
    'auth.verify.invalid_title': 'Lien invalide ou expiré',
    'auth.verify.invalid_desc':
      "Ce lien de vérification est invalide ou a expiré. Demande-en un nouveau ci-dessous.",
    'auth.forgot.title': 'Mot de passe oublié ?',
    'auth.forgot.subtitle':
      "Saisis ton adresse e-mail : on t'envoie un lien pour choisir un nouveau mot de passe.",
    'auth.forgot.submit': 'Envoyer le lien',
    'auth.forgot.submitting': 'Envoi en cours…',
    'auth.forgot.sent_title': 'Vérifie ta boîte mail',
    'auth.forgot.sent_desc':
      "Si un compte correspond à cette adresse, un e-mail avec un lien de réinitialisation vient d'être envoyé. Le lien expire dans 1 heure.",
    'auth.reset.title': 'Nouveau mot de passe',
    'auth.reset.subtitle': 'Choisis un nouveau mot de passe pour ton compte Breezy.',
    'auth.reset.new_password_label': 'Nouveau mot de passe',
    'auth.reset.confirm_password_label': 'Confirme le mot de passe',
    'auth.reset.submit': 'Réinitialiser le mot de passe',
    'auth.reset.submitting': 'Réinitialisation…',
    'auth.reset.err.mismatch': 'Les deux mots de passe ne correspondent pas.',
    'auth.reset.err.generic': 'La réinitialisation a échoué. Réessaie.',
    'auth.reset.success_title': 'Mot de passe réinitialisé !',
    'auth.reset.success_desc':
      'Ton mot de passe a été mis à jour et toutes tes sessions ont été déconnectées. Connecte-toi avec ton nouveau mot de passe.',
    'auth.reset.go_to_login': 'Se connecter',
    'auth.reset.invalid_title': 'Lien invalide ou expiré',
    'auth.reset.invalid_desc':
      'Ce lien de réinitialisation est invalide, a expiré ou a déjà été utilisé. Demande-en un nouveau.',
    'auth.reset.request_new': 'Demander un nouveau lien',

    // — Commun (toasts) —
    'common.action_failed': 'Action impossible',
    'common.delete_failed': 'Suppression impossible',

    // — Fil d'actualité —
    'feed.title': "Fil d'actualité",
    'feed.tab_for_you': 'Pour toi',
    'feed.tab_following': 'Abonnements',
    'feed.load_error': 'Impossible de charger le fil.',
    'feed.unavailable': 'Fil indisponible',
    'feed.empty_title': 'Aucun post pour le moment',
    'feed.empty_for_you': 'Soyez le premier à publier quelque chose sur Breezy.',
    'feed.empty_following':
      'Les posts des comptes que vous suivez apparaîtront ici. Abonnez-vous à des profils pour personnaliser ce fil.',
    'feed.filtered_empty_title': 'Tous les posts visibles sont filtrés',
    'feed.filtered_empty_msg':
      'Modifiez vos mots filtrés dans les paramètres pour les revoir dans le fil.',

    // — Composer —
    'composer.placeholder': 'Ça breez ? 🌴',
    'composer.add_image': 'Ajouter une image ou une vidéo',
    'composer.media_failed': "Échec de l'envoi du média.",
    'composer.media_max': 'Maximum {count} médias par post.',
    'composer.media_remove': 'Retirer le média',
    'composer.add_emoji': 'Ajouter un emoji',
    'composer.add_poll': 'Ajouter un sondage',
    'composer.pin_profile': 'Épingler sur mon profil',
    'composer.post_failed': 'Publication impossible',
    'composer.dialog_desc': 'Rédigez et publiez un nouveau post (280 caractères maximum).',
    'emoji.aria': 'Emoji {emoji}',

    // — Post (carte) —
    'post.delete': 'Supprimer',
    'post.more_options': "Plus d'options",
    'post.comment': 'Commenter',
    'post.repost': 'Reposter',
    'post.like': 'Aimer',
    'post.views': 'Vues',
    'post.share': 'Partager',
    'post.deleted': 'Post supprimé',

    // — Commentaires —
    'comment.reply': 'Répondre',
    'comment.placeholder': 'Écrire un commentaire…',
    'comment.reply_placeholder': 'Écrire une réponse…',
    'comment.load_error': 'Impossible de charger les commentaires.',
    'comment.empty': 'Aucun commentaire. Soyez le premier à réagir.',
    'comment.hide_replies': 'Masquer les réponses',
    'comment.view_replies_one': 'Voir les {count} réponse',
    'comment.view_replies_other': 'Voir les {count} réponses',
    'comment.view_more_replies': 'Voir plus de réponses',
    'comment.delete_aria': 'Supprimer le commentaire',
    'comment.submit_failed': 'Commentaire impossible',
    'comment.reply_failed': 'Réponse impossible',
    'comment.load_failed': 'Chargement impossible',

    // — Profil —
    'profil.not_found': 'Profil introuvable.',
    'profil.updated': 'Profil mis à jour',
    'profil.update_failed': 'Mise à jour impossible',
    'profil.save_failed': "Le profil n'a pas pu être enregistré.",
    'profil.unavailable': 'Profil indisponible',
    'profil.banned_title': 'Compte banni',
    'profil.banned_desc': 'Ce compte a été banni par un administrateur et n’est plus accessible.',
    'profil.back_aria': 'Retour au fil',
    'profil.posts_count_one': '{count} post',
    'profil.posts_count_other': '{count} posts',
    'profil.tab_posts': 'Posts',
    'profil.tab_replies': 'Réponses',
    'profil.tab_likes': "J'aime",
    'profil.empty_posts': 'Aucun post publié pour le moment.',
    'profil.empty_replies': 'Les réponses apparaîtront ici.',
    'profil.empty_likes': 'Les posts que vous aimez apparaîtront ici.',
    'profil.private_title': 'Ce compte est privé',
    'profil.private_message':
      'Suivez ce profil pour voir ses posts, ses réponses et ses mentions J’aime.',
    'profil.edit': 'Éditer le profil',
    'profil.born_on': 'Né(e) le {date}',
    'profil.joined': 'A rejoint en {date}',
    'profil.followers': 'Abonnés',
    'profil.following': 'Abonnements',

    // — Bouton de suivi —
    'follow.follow': 'Suivre',
    'follow.followed': 'Abonné',
    'follow.requested': 'En attente',
    'follow.unfollow': 'Ne plus suivre',
    'follow.private_unfollow_title': 'Ne plus suivre ce compte privé ?',
    'follow.private_unfollow_desc':
      'Vous perdrez l’accès à ses posts, ses réponses, ses mentions J’aime et ses listes d’abonnements.',
    'follow.private_unfollow_confirm': 'Ne plus suivre',
    'follow.remove_follower': 'Retirer',
    'follow.fail_title': 'Suivi impossible',
    'follow.unfail_title': 'Désabonnement impossible',
    'follow.remove_follower_fail_title': "Retrait impossible",
    'follow.fail_desc': 'Connectez-vous pour gérer vos abonnements.',

    // — Liste d'utilisateurs —
    'list.view_profile_aria': 'Voir le profil de {name}',

    // — Carte de survol profil —
    'profile_hover.follow': "S'abonner",
    'profile_hover.following': 'Abonné',
    'profile_hover.load_error': "Aperçu du profil indisponible.",
    'profile_hover.followed_by': 'Suivi par {names}',

    // — Modale des relations —
    'relations.title': 'Connexions',
    'relations.load_error': 'Impossible de charger la liste.',
    'relations.tab_followers': '{count} Abonnés',
    'relations.tab_following': '{count} Abonnements',
    'relations.empty_followers': 'Aucun abonné pour le moment.',
    'relations.empty_following': 'Aucun abonnement pour le moment.',
    'relations.private_locked':
      'Ce profil est privé. Les compteurs restent visibles, mais la liste détaillée est réservée aux abonnés.',

    // — Édition du profil —
    'editprofil.desc': 'Mettez à jour les informations visibles sur votre profil public.',
    'editprofil.change_banner': 'Changer la bannière',
    'editprofil.change_avatar': 'Changer la photo de profil',
    'editprofil.upload_failed': "Échec de l'envoi de l'image.",
    'editprofil.name_label': 'Nom',
    'editprofil.name_placeholder': 'Votre nom',
    'editprofil.name_max': '{count} caractères maximum.',
    'editprofil.name_locked': 'Le pseudo pourra être changé le {date}.',
    'editprofil.bio_label': 'Bio',
    'editprofil.bio_placeholder': 'Parlez de vous en quelques mots…',
    'editprofil.location_label': 'Localisation',
    'editprofil.location_placeholder': 'Ville, pays',
    'editprofil.website_label': 'Site web',
    'editprofil.birthdate_locked':
      "La date de naissance ne peut pas être changée une fois renseignée.",
    'editprofil.gender_locked': 'Le genre ne peut pas être changé une fois renseigné.',
    'editprofil.gender_none': 'Non renseigné',
    'editprofil.saving': 'Enregistrement...',

    // — Explorer —
    'explorer.search_placeholder': 'Rechercher un compte',
    'explorer.hint_before': 'Astuce : commencez par',
    'explorer.hint_after': 'pour chercher par identifiant.',
    'explorer.search_failed_title': 'Recherche impossible',
    'explorer.search_failed_msg': 'Réessayez dans un instant.',
    'explorer.empty_title': 'Rechercher sur Breezy',
    'explorer.empty_msg': 'Trouvez des comptes par nom ou par identifiant (@).',
    'explorer.history_title': 'Recherches récentes',
    'explorer.history_clear': "Effacer l'historique",
    'explorer.history_remove_one': "Retirer {name} de l'historique",
    'explorer.no_results': 'Aucun résultat',
    'explorer.no_results_handle': 'Aucun identifiant ne correspond à « {q} ».',
    'explorer.no_results_name': 'Aucun nom ne correspond à « {q} ».',

    // — Qui suivre —
    'who.title': 'Qui suivre',
    'who.empty': 'Aucune suggestion pour le moment.',
  },
  en: {
    // — Common —
    'common.user': 'User',
    'common.username_fallback': 'user',
    'common.not_connected': 'not signed in',
    'common.logout': 'Log out',
    'common.cancel': 'Cancel',
    'common.save': 'Save',
    'common.loading': 'Loading…',
    'common.retry': 'Retry',
    'common.close': 'Close',
    'media.download': 'Download image',
    'media.previous': 'Previous image',
    'media.next': 'Next image',
    'media.speed': 'Playback speed',
    'media.speed_normal': 'Normal',

    // — Navigation —
    'nav.feed': 'Feed',
    'nav.explore': 'Explore',
    'nav.notifications': 'Notifications',
    'nav.messages': 'Messages',
    'nav.profil': 'Profile',
    'nav.moderation': 'Moderation',
    'nav.admin': 'Administration',
    'nav.settings': 'Settings',
    'nav.home': 'Home',
    'nav.bookmarks': 'Bookmarks',
    'nav.open_menu': 'Open navigation menu',
    'nav.post': 'Breeze',
    'nav.compose': 'Compose a post',

    // — Visitor mode (signed out) —
    'visitor.login': 'Log in',
    'visitor.register': 'Sign up',
    'visitor.cta_aria': 'Log in or sign up',
    'visitor.prompt_title': 'Join Breezy',
    'visitor.prompt_desc':
      'Log in or create an account to like, comment, repost and follow people.',

    'bookmarks.title': 'Bookmarks',
    'bookmarks.all': 'All',
    'bookmarks.default_name': 'My bookmarks',
    'bookmarks.save': 'Save',
    'bookmarks.save_to': 'Save to a collection',
    'bookmarks.organize': 'Organize…',
    'bookmarks.saved_to': 'Saved to “{name}”',
    'bookmarks.removed': 'Removed from bookmarks',
    'bookmarks.add_aria': 'Add to bookmarks',
    'bookmarks.remove_aria': 'Remove from bookmarks',
    'bookmarks.new_collection': 'New collection',
    'bookmarks.new_collection_placeholder': 'Collection name',
    'bookmarks.create': 'Create',
    'bookmarks.no_collections': 'No collection yet.',
    'bookmarks.manage': 'Manage collection',
    'bookmarks.rename': 'Rename',
    'bookmarks.delete': 'Delete',
    'bookmarks.delete_title': 'Delete collection',
    'bookmarks.delete_confirm': 'Posts will leave this collection (they stay published).',
    'bookmarks.empty_title': 'No bookmarks',
    'bookmarks.empty_message': 'Save posts to find them here.',
    'bookmarks.empty_collection': 'This collection is empty.',
    'bookmarks.load_error': 'Could not load bookmarks.',
    'nav.search': 'Search',

    // — Roles —
    'role.user': 'User',
    'role.moderator': 'Moderator',
    'role.administrator': 'Administrator',

    // — Theme switch —
    'theme.title': 'Theme',
    'theme.appearance': 'Appearance',
    'theme.toggle_aria': 'Switch between light and dark mode',
    'theme.system': 'System mode',
    'theme.on': 'On',
    'theme.off': 'Off',
    // — Custom theme —
    'theme.customize': 'Customize theme',
    'theme.customize_hint': 'Pick your Breezy colors.',
    'theme.target_background': 'Background',
    'theme.target_background_hint': 'App background color',
    'theme.target_text': 'Text color',
    'theme.target_text_hint': '“For you”, “Following”…',
    'theme.target_primary': 'Action buttons',
    'theme.target_primary_hint': '“Breeze”, “Follow”…',
    'theme.reset': 'Reset',
    'theme.back': 'Back',
    'theme.hex': 'Hex code',
    'theme.brightness': 'Brightness',
    'theme.color_wheel_aria': 'Color selection wheel',

    // — Language switch —
    'lang.title': 'Language',
    'lang.select_aria': 'Choose language',

    // — Settings —
    'settings.title': 'Settings',
    'settings.account_title': 'Account settings',
    'settings.account_desc':
      'Customize your Breezy experience.',
    'filters.title': 'Muted words',
    'filters.desc':
      "Hide feed posts containing a word or phrase you don't want to see.",
    'filters.placeholder': 'E.g. one piece',
    'filters.input_aria': 'Word or phrase to mute',
    'filters.add': 'Add filter',
    'filters.remove_aria': 'Remove filter {word}',
    'filters.empty': 'No muted words yet.',

    // — Legal pages —
    'legal.badge': 'Legal information',
    'legal.updated': 'Last updated: {date}',
    'legal.back': 'Back to Breezy',
    'legal.section': 'Legal',
    'legal.mentions': 'Legal notice',
    'legal.cgu': 'Terms of Use',
    'legal.confidentialite': 'Privacy',

    // — Notifications —
    'notifications.title': 'Notifications',
    'notifications.unread_aria': '{count} unread notifications',
    'notifications.empty': 'No notifications yet.',
    'notifications.load_more': 'Show more',
    'notifications.like_one': '{name} liked your post',
    'notifications.like_other': '{name} and {count} others liked your post',
    'notifications.comment_one': '{name} commented on your post',
    'notifications.comment_other': '{name} and {count} others commented on your post',
    'notifications.reply_one': '{name} replied to your comment',
    'notifications.reply_other': '{name} and {count} others replied to your comment',
    'notifications.repost_one': '{name} reposted your post',
    'notifications.repost_other': '{name} and {count} others reposted your post',
    'notifications.quote': '{name} quoted your post',
    'notifications.mention': '{name} mentioned you',
    'notifications.message_mention_one': '{name} mentioned you in a message',
    'notifications.message_mention_other': '{name} mentioned you in {count} messages',
    'notifications.follow_one': '{name} followed you',
    'notifications.follow_other': '{name} and {count} others followed you',

    // — Mentions (@handle) —
    'mentions.view_in_search': 'View in search',
    'mentions.user_not_found': 'Account not found',
    'notifications.follow_request': '{name} requested to follow you',
    'notifications.follow_request_accepted': '{name} accepted your request',
    'notifications.follow_request_accept_confirm': "You accepted {name}'s request",
    'notifications.follow_request_accept_confirm_prefix': "You accepted the request from",
    'notifications.post_purge_warning':
      'One of your tweets removed by moderation will be permanently deleted in about a month.',
    'notifications.accept': 'Accept',
    'notifications.reject': 'Reject',

    // — Post detail —
    'common.back': 'Back',
    'post.detail_title': 'Post',
    'post.not_found': 'Post not found.',

    // — Stub pages (empty states) —
    'notifications.heading': 'Nothing yet',
    'notifications.desc': 'Your notifications (likes, follows, mentions) will show up here.',
    'messages.title': 'Messages',
    'messages.heading': 'No conversations',
    'messages.empty_desc': 'Start a chat with the + button.',
    'messages.new': 'New',
    'messages.new_dm': 'New message',
    'messages.new_dm_desc': 'Search for someone to start an end-to-end encrypted conversation.',
    'messages.new_group': 'New group',
    'messages.new_group_desc':
      'Name the group and add members. The name and messages are end-to-end encrypted.',
    'messages.new_community': 'Create a community',
    'messages.discover': 'Discover communities',
    'messages.discover_desc':
      'Join public communities. You enter as read-only until you are promoted.',
    'messages.create_group': 'Create group',
    'messages.create_community': 'Create',
    'messages.group_name_placeholder': 'Group name',
    'messages.community_name_placeholder': 'Community name',
    'messages.community_note':
      'The name is public and messages are readable by an administrator (semi-public community).',
    'messages.community_admin_note':
      'Semi-public community: messages can be read by an administrator.',
    'messages.search_user_placeholder': 'Search for someone (@ for the handle)…',
    'messages.search_community_placeholder': 'Search for a community…',
    'messages.no_users': 'No user found.',
    'messages.no_communities': 'No community found.',
    'messages.select_title': 'Your messages',
    'messages.select_desc': 'Select a conversation or start a new one.',
    'messages.composer_placeholder': 'Write a message…',
    'messages.send': 'Send',
    'messages.edit': 'Edit',
    'messages.editing': 'Editing message',
    'messages.cancel_edit': 'Cancel edit',
    'messages.save_edit': 'Save',
    'messages.edited': 'edited',
    'messages.hide_original': 'hide original',
    'messages.add_attachment': 'Attach an image or video',
    'messages.open_image': 'View image larger',
    'messages.attachment_failed': 'Attachment unavailable',
    'messages.attachment_preview': 'Attachment',
    'messages.back': 'Back',
    'messages.info': 'Details',
    'messages.members': 'Members',
    'messages.members_count': '{count} members',
    'messages.invite': 'Invite',
    'messages.rename': 'Rename',
    'messages.leave': 'Leave',
    'messages.delete': 'Delete',
    'messages.remove_member': 'Remove',
    'messages.promote': 'Promote (can post)',
    'messages.demote': 'Demote (read-only)',
    'messages.join': 'Join',
    'messages.joined': 'Joined',
    'messages.you': 'you',
    'messages.message_action': 'Message',
    'messages.read_only': 'Read-only — you cannot post in this community.',
    'messages.key_missing':
      'Key unavailable on this device: this conversation was created on another device.',
    'messages.decrypt_failed': 'Encrypted message (key unavailable)',
    'messages.passphrase.setup_title': 'Protect your messages',
    'messages.passphrase.setup_desc':
      'Set a memorable passphrase to access your private messages on your other devices. It encrypts your key: we cannot recover it if you forget it.',
    'messages.passphrase.unlock_title': 'Unlock your messages',
    'messages.passphrase.unlock_desc':
      'Enter the passphrase you set on your first device to decrypt your private messages here.',
    'messages.passphrase.field': 'Passphrase',
    'messages.passphrase.field_placeholder': 'Your passphrase',
    'messages.passphrase.confirm_field': 'Confirm passphrase',
    'messages.passphrase.show': 'Show passphrase',
    'messages.passphrase.hide': 'Hide passphrase',
    'messages.passphrase.setup_cta': 'Set my passphrase',
    'messages.passphrase.unlock_cta': 'Unlock',
    'messages.passphrase.working': 'Please wait…',
    'messages.passphrase.strength.weak': 'Weak',
    'messages.passphrase.strength.fair': 'Fair',
    'messages.passphrase.strength.good': 'Good',
    'messages.passphrase.strength.strong': 'Strong',
    'messages.passphrase.too_weak':
      'Too simple: aim for at least 10 characters and a mix of words/numbers/symbols.',
    'messages.passphrase.mismatch': 'The two passphrases do not match.',
    'messages.passphrase.wrong': 'Incorrect passphrase.',
    'messages.passphrase.setup_failed': 'Could not save the passphrase. Please try again.',
    'messages.passphrase.unlock_failed': 'Could not unlock. Check your passphrase.',
    'messages.passphrase.warning': 'Keep it safe: without it, your messages stay inaccessible elsewhere.',
    'messages.no_messages_title': 'No messages',
    'messages.no_messages_desc': 'Send the first message 👋',
    'messages.new_messages_divider': 'New messages',
    'messages.you_prefix': 'You: {text}',
    'messages.mentioned_you': '{name} mentioned you',
    'messages.unread_aria': 'Unread messages',
    'messages.badge_aria': '{count} conversations with unread messages',
    'messages.actions_aria': 'Conversation actions',
    'messages.pin': 'Pin',
    'messages.unpin': 'Unpin',
    'messages.mute': 'Mute',
    'messages.unmute': 'Unmute',
    'messages.muted_aria': 'Muted conversation',
    'messages.delete_for_me': 'Delete for me',
    'messages.delete_failed': 'Could not delete.',
    'messages.search_in_conversation': 'Search this conversation…',
    'messages.search_no_results': 'No message found',
    'messages.type_dm': 'Direct message',
    'messages.type_group': 'Group',
    'messages.type_community': 'Community',
    'messages.role_owner': 'Owner',
    'messages.role_admin': 'Admin',
    'messages.role_member': 'Member',
    'messages.role_talker': 'Speaker',
    'messages.role_viewer': 'Viewer',
    'messages.load_failed': 'Could not load messages.',
    'messages.send_failed': 'Failed to send the message.',
    'messages.edit_failed': 'Failed to edit the message.',
    'messages.dm_failed': 'Could not start the conversation.',
    'messages.group_failed': 'Could not create the group.',
    'messages.community_failed': 'Could not create the community.',
    'messages.join_failed': 'Could not join the community.',
    'messages.action_failed': 'Action failed.',
    'admin.heading': 'Breezy administration',
    'admin.desc': 'User and role management will live here.',
    'admin.subtitle': 'Manage accounts, roles and access.',
    'admin.search_placeholder': 'Search by email',
    'admin.empty': 'No account found.',
    'admin.error': 'Could not load accounts.',
    'admin.status_active': 'Active',
    'admin.status_banned': 'Banned',
    'admin.action_ban': 'Ban',
    'admin.action_unban': 'Reinstate',
    'admin.you': 'You',
    'admin.role_changed': 'Role updated: {role}',
    'admin.banned_toast': 'Account banned.',
    'admin.unbanned_toast': 'Account reinstated.',
    'admin.action_failed': 'Action failed.',
    'admin.access_denied': 'Access denied',
    'admin.access_denied_desc': 'This page is restricted to administrators.',
    'admin.role_note': 'A role change takes effect the next time the user signs in (token refresh).',
    'moderation.heading': 'Moderation center',
    'moderation.desc': 'Reports and moderation actions will show up here.',
    'moderation.subtitle': 'Keep order on Breezy: removed tweets and accounts.',
    'moderation.tab_posts': 'Deleted tweets',
    'moderation.tab_accounts': 'Accounts',
    'moderation.posts_empty': 'No tweets in the trash.',
    'moderation.posts_error': 'Could not load the trash.',
    'moderation.restore': 'Restore',
    'moderation.purge': 'Delete permanently',
    'moderation.purge_title': 'Permanently delete this tweet?',
    'moderation.purge_desc':
      'This cannot be undone: the tweet and its likes, comments and reposts will be erased for good.',
    'moderation.removed_by': 'Removed by {who} · {when}',
    'moderation.purge_scheduled': 'Scheduled for permanent deletion on {when}',
    'moderation.expiring_soon': 'Expiring soon',
    'moderation.restored_toast': 'Tweet restored.',
    'moderation.purged_toast': 'Tweet permanently deleted.',
    'moderation.access_denied': 'Access denied',
    'moderation.access_denied_desc': 'This page is reserved for moderators and administrators.',
    'moderation.ban_restricted': 'A moderator can only ban a user.',
    'moderation.banned_since': 'Banned {when}',
    'moderation.account_delete': 'Delete account',
    'moderation.account_delete_soon': 'Permanent account deletion (GDPR) — coming soon.',
    'moderation.account_delete_title': 'Permanently delete this account?',
    'moderation.account_delete_desc':
      'Irreversible GDPR erasure: profile, tweets, messages, media and relationships will be deleted across all services. Type “{username}” to confirm.',
    'moderation.account_delete_confirm_label': 'Username to confirm',
    'moderation.account_deleted_toast': 'Account permanently deleted.',
    'moderation.account_delete_failed': 'Deletion failed (partial erasure possible).',
    'admin.infra_subtitle': 'Infrastructure and service monitoring.',
    'admin.infra_heading': 'Service status',
    'admin.infra_desc':
      'Docker container health and service uptime will appear here soon.',
    'admin.monitoring_error': 'Could not load service status.',
    'admin.monitoring_updated': 'Updated {when}',
    'admin.status_up': 'Online',
    'admin.status_down': 'Offline',
    'admin.latency': 'Latency',
    'admin.uptime': 'Uptime',
    'admin.uptime_d': 'd',
    'admin.uptime_h': 'h',
    'admin.uptime_m': 'm',
    'admin.uptime_s': 's',

    // — Search / Trends (right column) —
    'search.placeholder': 'Search',
    'trends.title': 'Trends',
    'trends.trending': 'Trending',
    'trends.t1.category': 'Technology',
    'trends.t1.posts': '12.4K posts',
    'trends.t2.category': 'Dev',
    'trends.t2.posts': '8.1K posts',
    'trends.t3.category': 'Cloud',
    'trends.t3.posts': '5.6K posts',

    // — Mobile navigation —
    'nav.main_aria': 'Main navigation',
    'post.create_aria': 'Create a post',

    // — 404 —
    'notfound.message': "This page doesn't exist.",
    'notfound.back': 'Back to home',

    // — Authentication (shared) —
    'auth.search_aria': 'Search',
    'auth.email_label': 'Email address',
    'auth.email_placeholder': 'you@example.com',
    'auth.password_label': 'Password',
    'auth.show_password': 'Show password',
    'auth.hide_password': 'Hide password',
    'auth.err.email_required': 'Email address is required.',
    'auth.err.email_invalid': 'Enter a valid email address.',
    'auth.err.email_max': 'Email address is limited to 50 characters.',
    'auth.err.identifier_required': 'Email address or username is required.',
    'auth.err.password_required': 'Password is required.',
    'auth.err.password_min': 'Password must be at least 8 characters.',
    'auth.err.network': "Couldn't reach the API. Try again in a moment.",

    // — Login —
    'auth.login.demo.kicker': 'Live feed',
    'auth.login.demo.heading': 'Find your world again.',
    'auth.login.demo.post1': 'New playlist, new debates, same Breezy energy.',
    'auth.login.demo.views': '18.4K views',
    'auth.login.demo.joined': 'Noa joined the conversation',
    'auth.login.demo.joined_sub': "Discover tonight's rising topics.",
    'auth.login.demo.rank': '#{n} on Breezy',
    'auth.login.demo.interactions': '17 new interactions',
    'auth.login.demo.interactions_sub': 'Your feed is waiting, fresh and alive.',
    'auth.login.badge': 'Sign in to the network',
    'auth.login.title': 'Pick up your feed where you left off.',
    'auth.login.subtitle':
      'Sign in to Breezy and find your messages, your posts and the conversations that move.',
    'auth.login.identifier_label': 'Email address or Username',
    'auth.login.identifier_placeholder': 'you@example.com or username',
    'auth.login.forgot': 'Forgot password?',
    'auth.login.submit': 'Sign in',
    'auth.login.submitting': 'Signing in…',
    'auth.login.or': 'Or sign in with',
    'auth.login.no_account': 'No account yet?',
    'auth.login.create_account': 'Create an account',
    'auth.login.failed': 'Sign-in failed. Check your credentials.',
    'auth.login.account_disabled':
      'Account disabled — please contact Breezy support.',

    // — OAuth —
    'auth.oauth.loading': 'Signing in…',
    'auth.oauth.error': 'Sign-in failed. Try again from the sign-in page.',
    'auth.oauth.back_to_login': 'Back to sign in',

    // — Register —
    'auth.register.demo.kicker': 'New to Breezy',
    'auth.register.demo.heading': 'Create your space.',
    'auth.register.demo.you': 'You',
    'auth.register.demo.now': 'now',
    'auth.register.demo.post1':
      'First post, first vibe, and already a whole community to meet.',
    'auth.register.demo.welcome': 'Welcome',
    'auth.register.demo.opens': 'Breezy opens up the feed',
    'auth.register.demo.opens_sub': 'Pick your handle and start posting.',
    'auth.register.demo.to_join': 'To join',
    'auth.register.demo.community': 'Community #{n}',
    'auth.register.demo.comm1': 'Creators',
    'auth.register.demo.comm2': 'CESI Campus',
    'auth.register.demo.comm3': 'Distributed Dev',
    'auth.register.demo.alive': 'Your account comes to life',
    'auth.register.demo.alive_sub': 'Profile, posts and conversations in seconds.',
    'auth.register.badge': 'New Breezy profile',
    'auth.register.title': 'Join Breezy and start posting.',
    'auth.register.subtitle':
      'Create your account, pick your username and step into the feed.',
    'auth.register.username_label': 'Username',
    'auth.register.username_tooltip':
      'The username can be changed after account creation, then once every 14 days.',
    'auth.register.birthdate_label': 'Date of birth',
    'auth.register.birthdate_tooltip':
      "Your date of birth can't be changed once the account is created.",
    'auth.register.gender_label': 'Gender',
    'auth.register.gender_male': 'Male',
    'auth.register.gender_female': 'Female',
    'auth.register.password_confirm_label': 'Password confirmation',
    'auth.register.show_password_confirm': 'Show password confirmation',
    'auth.register.hide_password_confirm': 'Hide password confirmation',
    'auth.register.password_help':
      '8 chars min., uppercase, lowercase, number and special character.',
    'auth.register.submit': 'Create my account',
    'auth.register.submitting': 'Creating account…',
    'auth.register.or': 'Or create my account with',
    'auth.register.have_account': 'Already have an account?',
    'auth.register.err.username_required': 'Username is required.',
    'auth.register.err.username_format':
      '3 to 24 characters: letters, numbers and underscore (_) only.',
    'auth.register.err.username_max': 'Username is limited to 24 characters.',
    'auth.register.err.username_reserved': "This username isn't allowed.",
    'auth.register.err.username_taken': 'This username is already taken.',
    'auth.register.err.username_check':
      "Couldn't check the username. Try again in a moment.",
    'auth.register.err.birthdate_required': 'Date of birth is required.',
    'auth.register.err.birthdate_invalid': 'Enter a valid date of birth.',
    'auth.register.err.birthdate_future': "Date of birth can't be in the future.",
    'auth.register.err.age': 'You must be at least 13 to sign up.',
    'auth.register.err.gender_required': 'Choose a gender.',
    'auth.register.err.password_max': 'Password is limited to 250 characters.',
    'auth.register.err.password_format':
      'At least 8 characters, one uppercase, one lowercase, one number and one special character.',
    'auth.register.err.confirm_required': 'Confirm your password.',
    'auth.register.err.confirm_max': 'The confirmation is limited to 250 characters.',
    'auth.register.err.confirm_mismatch': "Passwords don't match.",
    'auth.register.err.failed': 'Sign-up failed. Check the information entered.',

    // — Onboarding (finalize OAuth accounts without a profile) —
    'onboarding.title': 'Welcome to Breezy 👋',
    'onboarding.subtitle':
      'Pick your username and enter your date of birth to finish setting up your account.',
    'onboarding.username_label': 'Username',
    'onboarding.username_placeholder': 'e.g. felipe',
    'onboarding.username_hint': '3 to 24 characters: letters, numbers and underscore (_).',
    'onboarding.username_checking': 'Checking availability…',
    'onboarding.username_available': 'Available ✓',
    'onboarding.birthdate_label': 'Date of birth',
    'onboarding.birthdate_disclaimer':
      "Under 18, sensitive (NSFW) content will be hidden. Your date of birth can't be changed after confirmation.",
    'onboarding.submit': 'Confirm and continue',
    'onboarding.submitting': 'Saving…',
    'onboarding.err.generic': 'Something went wrong. Try again in a moment.',

    // — Email verification —
    'auth.check_email.title': 'Check your inbox',
    'auth.check_email.subtitle':
      'A verification link has just been sent to {email}. Click it to activate your account.',
    'auth.check_email.subtitle_generic':
      'A verification link has just been sent to your address. Click it to activate your account.',
    'auth.check_email.hint':
      'Remember to check your spam folder. The link expires in 24 hours.',
    'auth.check_email.resend': 'Resend verification link',
    'auth.check_email.back_to_login': 'Back to sign in',
    'auth.verify.login_blocked':
      "Your email address isn't verified yet. Check your inbox to activate your account.",
    'auth.verify.resend_cta': 'Resend verification email',
    'auth.verify.resending': 'Sending…',
    'auth.verify.resend_done':
      'If an unverified account matches this address, an email has just been sent.',
    'auth.verify.resend_label': 'Your email address',
    'auth.verify.loading_title': 'Verifying…',
    'auth.verify.loading_desc': 'One moment, we are confirming your email address.',
    'auth.verify.success_title': 'Email verified!',
    'auth.verify.success_desc':
      'Your account is active and you are now signed in. Welcome to Breezy!',
    'auth.verify.go_to_app': 'Enter Breezy',
    'auth.verify.invalid_title': 'Invalid or expired link',
    'auth.verify.invalid_desc':
      'This verification link is invalid or has expired. Request a new one below.',
    'auth.forgot.title': 'Forgot your password?',
    'auth.forgot.subtitle':
      "Enter your email address and we'll send you a link to choose a new password.",
    'auth.forgot.submit': 'Send the link',
    'auth.forgot.submitting': 'Sending…',
    'auth.forgot.sent_title': 'Check your inbox',
    'auth.forgot.sent_desc':
      'If an account matches this address, an email with a reset link has just been sent. The link expires in 1 hour.',
    'auth.reset.title': 'New password',
    'auth.reset.subtitle': 'Choose a new password for your Breezy account.',
    'auth.reset.new_password_label': 'New password',
    'auth.reset.confirm_password_label': 'Confirm password',
    'auth.reset.submit': 'Reset password',
    'auth.reset.submitting': 'Resetting…',
    'auth.reset.err.mismatch': 'The two passwords do not match.',
    'auth.reset.err.generic': 'Reset failed. Please try again.',
    'auth.reset.success_title': 'Password reset!',
    'auth.reset.success_desc':
      'Your password has been updated and all your sessions have been signed out. Sign in with your new password.',
    'auth.reset.go_to_login': 'Sign in',
    'auth.reset.invalid_title': 'Invalid or expired link',
    'auth.reset.invalid_desc':
      'This reset link is invalid, has expired, or has already been used. Request a new one.',
    'auth.reset.request_new': 'Request a new link',

    // — Common (toasts) —
    'common.action_failed': 'Action failed',
    'common.delete_failed': "Couldn't delete",

    // — Feed —
    'feed.title': 'Feed',
    'feed.tab_for_you': 'For you',
    'feed.tab_following': 'Following',
    'feed.load_error': "Couldn't load the feed.",
    'feed.unavailable': 'Feed unavailable',
    'feed.empty_title': 'No posts yet',
    'feed.empty_for_you': 'Be the first to post something on Breezy.',
    'feed.empty_following':
      'Posts from accounts you follow will show up here. Follow some profiles to personalize this feed.',
    'feed.filtered_empty_title': 'All visible posts are muted',
    'feed.filtered_empty_msg':
      'Edit your muted words in settings to show them in the feed again.',

    // — Composer —
    'composer.placeholder': "What's breezing? 🌴",
    'composer.add_image': 'Add an image or video',
    'composer.media_failed': 'Media upload failed.',
    'composer.media_max': 'Up to {count} media per post.',
    'composer.media_remove': 'Remove media',
    'composer.add_emoji': 'Add an emoji',
    'composer.add_poll': 'Add a poll',
    'composer.pin_profile': 'Pin to my profile',
    'composer.post_failed': "Couldn't post",
    'composer.dialog_desc': 'Write and publish a new post (280 characters max).',
    'emoji.aria': 'Emoji {emoji}',

    // — Post (card) —
    'post.delete': 'Delete',
    'post.more_options': 'More options',
    'post.comment': 'Comment',
    'post.repost': 'Repost',
    'post.like': 'Like',
    'post.views': 'Views',
    'post.share': 'Share',
    'post.deleted': 'Post deleted',

    // — Comments —
    'comment.reply': 'Reply',
    'comment.placeholder': 'Write a comment…',
    'comment.reply_placeholder': 'Write a reply…',
    'comment.load_error': "Couldn't load comments.",
    'comment.empty': 'No comments yet. Be the first to react.',
    'comment.hide_replies': 'Hide replies',
    'comment.view_replies_one': 'View {count} reply',
    'comment.view_replies_other': 'View {count} replies',
    'comment.view_more_replies': 'View more replies',
    'comment.delete_aria': 'Delete comment',
    'comment.submit_failed': "Couldn't comment",
    'comment.reply_failed': "Couldn't reply",
    'comment.load_failed': "Couldn't load",

    // — Profile —
    'profil.not_found': 'Profile not found.',
    'profil.updated': 'Profile updated',
    'profil.update_failed': "Couldn't update",
    'profil.save_failed': "The profile couldn't be saved.",
    'profil.unavailable': 'Profile unavailable',
    'profil.banned_title': 'Account banned',
    'profil.banned_desc': 'This account has been banned by an administrator and is no longer available.',
    'profil.back_aria': 'Back to feed',
    'profil.posts_count_one': '{count} post',
    'profil.posts_count_other': '{count} posts',
    'profil.tab_posts': 'Posts',
    'profil.tab_replies': 'Replies',
    'profil.tab_likes': 'Likes',
    'profil.empty_posts': 'No posts yet.',
    'profil.empty_replies': 'Replies will show up here.',
    'profil.empty_likes': 'Posts you like will show up here.',
    'profil.private_title': 'This account is private',
    'profil.private_message':
      'Follow this profile to see their posts, replies, and likes.',
    'profil.edit': 'Edit profile',
    'profil.born_on': 'Born on {date}',
    'profil.joined': 'Joined {date}',
    'profil.followers': 'Followers',
    'profil.following': 'Following',

    // — Follow button —
    'follow.follow': 'Follow',
    'follow.followed': 'Following',
    'follow.requested': 'Pending',
    'follow.unfollow': 'Unfollow',
    'follow.private_unfollow_title': 'Unfollow this private account?',
    'follow.private_unfollow_desc':
      'You will lose access to their posts, replies, likes, and follow lists.',
    'follow.private_unfollow_confirm': 'Unfollow',
    'follow.remove_follower': 'Remove',
    'follow.fail_title': "Couldn't follow",
    'follow.unfail_title': "Couldn't unfollow",
    'follow.remove_follower_fail_title': "Couldn't remove follower",
    'follow.fail_desc': 'Sign in to manage your follows.',

    // — User list —
    'list.view_profile_aria': "View {name}'s profile",

    // — Profile hover card —
    'profile_hover.follow': 'Follow',
    'profile_hover.following': 'Following',
    'profile_hover.load_error': "Profile preview isn't available.",
    'profile_hover.followed_by': 'Followed by {names}',

    // — Relations dialog —
    'relations.title': 'Connections',
    'relations.load_error': "Couldn't load the list.",
    'relations.tab_followers': '{count} Followers',
    'relations.tab_following': '{count} Following',
    'relations.empty_followers': 'No followers yet.',
    'relations.empty_following': 'Not following anyone yet.',
    'relations.private_locked':
      'This profile is private. Counts stay visible, but the detailed list is only available to followers.',

    // — Profile editing —
    'editprofil.desc': 'Update the information shown on your public profile.',
    'editprofil.change_banner': 'Change banner',
    'editprofil.change_avatar': 'Change profile picture',
    'editprofil.upload_failed': 'Image upload failed.',
    'editprofil.name_label': 'Name',
    'editprofil.name_placeholder': 'Your name',
    'editprofil.name_max': '{count} characters max.',
    'editprofil.name_locked': 'Your handle can be changed on {date}.',
    'editprofil.bio_label': 'Bio',
    'editprofil.bio_placeholder': 'Tell us about yourself in a few words…',
    'editprofil.location_label': 'Location',
    'editprofil.location_placeholder': 'City, country',
    'editprofil.website_label': 'Website',
    'editprofil.birthdate_locked': "Your date of birth can't be changed once set.",
    'editprofil.gender_locked': "Gender can't be changed once set.",
    'editprofil.gender_none': 'Not specified',
    'editprofil.saving': 'Saving...',

    // — Explore —
    'explorer.search_placeholder': 'Search for an account',
    'explorer.hint_before': 'Tip: start with',
    'explorer.hint_after': 'to search by handle.',
    'explorer.search_failed_title': 'Search failed',
    'explorer.search_failed_msg': 'Try again in a moment.',
    'explorer.empty_title': 'Search on Breezy',
    'explorer.empty_msg': 'Find accounts by name or handle (@).',
    'explorer.history_title': 'Recent searches',
    'explorer.history_clear': 'Clear history',
    'explorer.history_remove_one': 'Remove {name} from history',
    'explorer.no_results': 'No results',
    'explorer.no_results_handle': 'No handle matches “{q}”.',
    'explorer.no_results_name': 'No name matches “{q}”.',

    // — Who to follow —
    'who.title': 'Who to follow',
    'who.empty': 'No suggestions yet.',
  },
}

/**
 * Traduit une clé pour une locale donnée, avec interpolation `{param}` optionnelle.
 *
 * Repli en cascade : locale demandée → DEFAULT_LOCALE → clé brute (pour repérer
 * une clé manquante en dev sans casser le rendu).
 */
export function translate(
  locale: Locale,
  key: string,
  params?: Record<string, string | number>,
): string {
  const table = messages[locale] ?? messages[DEFAULT_LOCALE]
  const template = table[key] ?? messages[DEFAULT_LOCALE][key] ?? key
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in params ? String(params[name]) : match,
  )
}
