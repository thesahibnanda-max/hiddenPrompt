import { Github } from "lucide-react";

const REPO_URL = "https://github.com/thesahibnanda-max/hiddenPrompt";

export function GithubSourceLink() {
  return (
    <a
      href={REPO_URL}
      target="_blank"
      rel="noopener noreferrer"
      title="View source on GitHub"
      aria-label="View source code on GitHub"
      className="fixed bottom-4 left-4 z-40 flex h-12 w-12 items-center justify-center rounded-full border border-neon-cyan/40 bg-void-2/80 text-neon-cyan shadow-neon backdrop-blur-md transition-transform hover:scale-105"
    >
      <Github size={20} />
    </a>
  );
}
