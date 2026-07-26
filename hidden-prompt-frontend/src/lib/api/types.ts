// DTOs mirror the backend's wire JSON exactly (pkg/handler/models.go, pkg/models/*.go).
// Field names/casing are load-bearing — do not "clean them up".

// ---- Requests ----
export interface SignUpRequest {
  email: string;
  password: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface DecideVerificationRequest {
  otp: string;
}

// NOTE: no `email` field here — the backend derives the caller's identity
// server-side from the X-Auth-Token header. This is the actual wire body,
// not the internal Go service-layer DTO (which does carry Email).
export interface AddUserPromptRequest {
  puzzle_id: string;
  user_prompt: string;
}

// ---- Responses ----
export interface SuccessResponse {
  message: string;
}

export interface LoginResponse {
  message: string;
  is_verified: boolean;
}

export interface PuzzleElement {
  puzzle_id: string;
  canonical_answer: string;
}

export type MessageRole = "user" | "assistant";

export interface MessageElement {
  role: MessageRole;
  message: string;
  timestamp: string; // RFC3339
}

export interface GuessMetrics {
  embedding_dot_product: number;
  cosine_similarity_score: number;
  llm_similarity_score: number;
  final_similarity_score: number;
  considered_won: boolean;
}

export interface PuzzleDetails {
  puzzle_id: string;
  canonical_answer: string;
  hidden_prompt?: string; // absent until won
  messages: MessageElement[];
  max_intent_similarity_percentage: number;
  user_win_status: boolean;
  latest_guess_metrics?: GuessMetrics; // only populated on POST /puzzle/guess responses
  created_at: string;
  updated_at: string;
}

export interface HealthResponse {
  status: "ok" | "unhealthy";
  errors?: Partial<Record<"mysql" | "postgresql" | "redis" | "mongodb", string>>;
}

export interface Track {
  id: string;
  title: string;
  artists: string[];
  genre: string[];
  wave_audio_link: string;
  artwork_link: string;
  created_at: string;
}

// ---- Error shape ----
export interface ApiErrorBody {
  show_response_as_is?: boolean; // absent on router-level 404/405 fallback bodies
  error_message: string;
  status_code?: number;
}
