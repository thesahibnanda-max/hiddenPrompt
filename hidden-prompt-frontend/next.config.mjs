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
};

export default nextConfig;
