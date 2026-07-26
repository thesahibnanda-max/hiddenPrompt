export function pick(options: string[]): string {
  return options[Math.floor(Math.random() * options.length)] ?? "";
}
