import { NextRequest, NextResponse } from "next/server";

// Server-side proxy to the plain-HTTP backend, since browsers block HTTPS
// pages from calling HTTP endpoints directly (mixed content). This replaces
// a next.config.mjs rewrite, which broke on Vercel: its routing manifest
// runs destinations through path-to-regexp, and a literal ":8080" in the
// destination URL gets parsed as a named route param instead of a port.
const BACKEND_URL = "http://80.225.227.90:8080";

const FORWARD_RESPONSE_HEADERS = ["content-type", "x-auth-token", "x-time-taken", "retry-after"];

async function proxy(req: NextRequest, path: string[]) {
  const target = `${BACKEND_URL}/${path.join("/")}${req.nextUrl.search}`;

  const authToken = req.headers.get("x-auth-token");
  const res = await fetch(target, {
    method: req.method,
    headers: {
      "Content-Type": req.headers.get("content-type") ?? "application/json",
      ...(authToken ? { "X-Auth-Token": authToken } : {}),
    },
    body: req.method === "GET" || req.method === "HEAD" ? undefined : await req.text(),
  });

  const body = await res.text();
  const headers = new Headers();
  for (const name of FORWARD_RESPONSE_HEADERS) {
    const value = res.headers.get(name);
    if (value) headers.set(name, value);
  }

  return new NextResponse(body, { status: res.status, headers });
}

export async function GET(req: NextRequest, { params }: { params: { path: string[] } }) {
  return proxy(req, params.path);
}

export async function POST(req: NextRequest, { params }: { params: { path: string[] } }) {
  return proxy(req, params.path);
}
