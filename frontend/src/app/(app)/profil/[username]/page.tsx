import { ProfilView } from '@/components/profil/profil-view'

interface PublicProfilPageProps {
  params: {
    username: string
  }
}

export default function PublicProfilPage({ params }: PublicProfilPageProps) {
  return <ProfilView username={params.username} />
}
