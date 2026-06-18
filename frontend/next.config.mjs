/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  // RIV-007 : ne pas divulguer la stack via l'en-tête `X-Powered-By: Next.js`.
  poweredByHeader: false,
}

export default nextConfig
