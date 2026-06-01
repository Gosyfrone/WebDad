// Composant racine — placeholder pour l'initialisation du mono-repo.
function App() {
  const gatewayUrl = import.meta.env.VITE_API_GATEWAY_URL ?? 'http://localhost:3000'

  return (
    <main style={{ fontFamily: 'sans-serif', padding: '2rem' }}>
      <h1>WebDad</h1>
      <p>Application distribuée en microservices.</p>
      <p>
        API Gateway : <code>{gatewayUrl}</code>
      </p>
    </main>
  )
}

export default App
