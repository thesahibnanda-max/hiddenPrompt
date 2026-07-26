/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  experimental: {
    // Tree-shakes per-icon/per-component imports from these two packages
    // instead of pulling the whole library into the client bundle - both
    // are used on every route (SynthwaveBackground, MuteButton, page
    // transitions), so this is a real bundle-size win, not a micro-opt.
    optimizePackageImports: ["lucide-react", "framer-motion"],
  },
  async rewrites() {
    // Proxies same-origin HTTPS calls to the plain-HTTP backend server-side,
    // since browsers block HTTPS pages from calling HTTP endpoints directly
    // (mixed content) - server-to-server isn't subject to that restriction.
    return [
      {
        source: "/api/backend/:path*",
        destination: "http://80.225.227.90:8080/:path*",
      },
    ];
  },
};

export default nextConfig;
