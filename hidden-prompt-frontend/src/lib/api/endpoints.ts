import { apiFetch } from "./client";
import type {
  AddUserPromptRequest,
  DecideVerificationRequest,
  HealthResponse,
  LoginRequest,
  LoginResponse,
  PuzzleDetails,
  PuzzleElement,
  SignUpRequest,
  SuccessResponse,
  Track,
} from "./types";

export const signUp = (body: SignUpRequest) =>
  apiFetch<SuccessResponse>("/users/signup", { method: "POST", body, auth: false });

export const logIn = (body: LoginRequest) =>
  apiFetch<LoginResponse>("/users/login", { method: "POST", body, auth: false });

export const initiateVerification = () =>
  apiFetch<SuccessResponse>("/users/verify/initiate", { method: "POST" });

export const decideVerification = (body: DecideVerificationRequest) =>
  apiFetch<SuccessResponse>("/users/verify/decide", { method: "POST", body });

export const createPuzzle = () => apiFetch<PuzzleElement[]>("/puzzle", { method: "POST" });

export const getPuzzles = () => apiFetch<PuzzleElement[]>("/puzzle");

export const getPuzzleDetail = (puzzleId: string) =>
  apiFetch<PuzzleDetails>("/puzzle/detail", { query: { puzzle_id: puzzleId } });

export const submitGuess = (body: AddUserPromptRequest) =>
  apiFetch<PuzzleDetails>("/puzzle/guess", { method: "POST", body });

export const getHealth = () => apiFetch<HealthResponse>("/health", { auth: false });

export const getTracks = () => apiFetch<Track[]>("/music", { auth: false });
