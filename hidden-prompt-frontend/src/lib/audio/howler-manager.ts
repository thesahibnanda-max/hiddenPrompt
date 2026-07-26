import { Howl, Howler } from "howler";
import type { Track } from "@/lib/api/types";
import { useAudioStore } from "@/stores/audio-store";
import { useMusicStore } from "@/stores/music-store";

class HowlerManager {
  private tracks: Track[] = [];
  private howls: Howl[] = [];
  private currentIndex = 0;
  private history: number[] = [];
  private started = false;
  // Set when start() is called (first user gesture, via AudioBootstrap)
  // before the track catalog has finished fetching - resolved the moment
  // setTracks() actually populates the playlist.
  private pendingStart = false;
  private unlockPrimed = false;

  // Constructs one throwaway, silent, data-URI-sourced Howl - never played,
  // never referenced again - purely to trigger Howler's own internal
  // AudioContext creation and its built-in gesture-based unlock listeners
  // (both happen as documented side effects of `new Howl(...)`, see
  // howler.js's Howl.init: it lazily creates Howler.ctx, then arms
  // Howler._unlockAudio() if that context now exists). Both only ever get
  // set up as a side effect of a Howl existing, so without this, they
  // wouldn't be armed until buildHowlsIfNeeded() runs - which can be well
  // after the user's first gesture if the track-list fetch is still in
  // flight, silently missing that gesture. A data URI never touches the
  // network, so this doesn't reintroduce the eager-download problem real
  // track Howls are deliberately kept lazy to avoid.
  primeUnlock() {
    if (this.unlockPrimed) return;
    this.unlockPrimed = true;
    // eslint-disable-next-line no-new
    new Howl({
      src: ["data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQAAAAA="],
      volume: 0,
    });
  }

  // Stores the fetched catalog, eagerly warms the first track's audio
  // (see preloadFirstTrack - the panel's loader gates on this one), then
  // kicks off every other track's audio in the background without
  // blocking anything (see preloadRemainingTracks).
  setTracks(tracks: Track[]) {
    this.tracks = tracks;
    this.preloadFirstTrack();
    this.preloadRemainingTracks();
    if (this.pendingStart) {
      this.pendingStart = false;
      this.start();
    }
  }

  // Warms track 0's audio immediately, independent of any gesture, so the
  // music panel has a concrete load/loaderror signal to gate its loading
  // message on ("ready to play" meaning the audio itself, not just the
  // catalog JSON) - the one track whose load time the user actually waits
  // on. A loaderror still resolves readiness - a broken first track must
  // not leave the panel stuck loading forever. An empty catalog has
  // nothing to preload, so it resolves immediately.
  private preloadFirstTrack() {
    if (this.tracks.length === 0) {
      useMusicStore.getState().setAudioReady(true);
      return;
    }
    const howl = this.ensureHowlBuilt(0);
    howl?.once("load", () => useMusicStore.getState().setAudioReady(true));
    howl?.once("loaderror", () => useMusicStore.getState().setAudioReady(true));
  }

  // Kicks off every remaining track's audio loading in parallel, in the
  // background, right after the catalog arrives - deliberately does not
  // gate or delay anything the panel waits on (only preloadFirstTrack
  // does that). By the time the user actually navigates to one of these,
  // it's very likely already loaded instead of starting a fresh download
  // at that moment. ensureHowlBuilt is idempotent, so this is a no-op for
  // any index a user has already navigated to before this runs.
  private preloadRemainingTracks() {
    for (let i = 1; i < this.tracks.length; i++) {
      this.ensureHowlBuilt(i);
    }
  }

  // Howler preloads/decodes its audio source on construction regardless of
  // whether playback actually starts, so every index beyond the eagerly
  // warmed first track stays lazy, only built the moment something
  // actually navigates to it (playIndex/start) - constructing every
  // track's audio up front would quietly download the whole catalog on
  // every page view, including tracks nobody ever plays.
  private ensureHowlBuilt(index: number): Howl | undefined {
    if (index < 0 || index >= this.tracks.length) return undefined;
    if (!this.howls[index]) {
      this.howls[index] = new Howl({
        src: [this.tracks[index].wave_audio_link],
        loop: false, // sequencing is manual (see onend) so the playlist shuffles instead of looping one track
        volume: 0.35,
        // Web Audio API mode (the default - no html5: true here) rather
        // than pooled native <audio> elements: Chrome's autoplay-unlock
        // policy is built around AudioContext.resume() on a trusted
        // gesture, which Howler already implements internally for this
        // mode. Pooled HTML5 audio unlock is a weaker path, and it's
        // what caused music to stay silent in Chrome Incognito (zero
        // persisted Media Engagement Index there, unlike a regular
        // profile) while working fine in a regular Edge window. This
        // still works fine for these remote (Supabase-hosted) sources
        // since they're served with a permissive
        // `Access-Control-Allow-Origin: *`, which is what Web Audio
        // API's fetch+decode path needs for a cross-origin source.
        onend: () => this.advance(),
      });
      this.registerContextResilienceListenersOnce();
    }
    return this.howls[index];
  }

  private contextListenersRegistered = false;

  // Chrome force-suspends the shared AudioContext when the tab/window is
  // backgrounded or minimized, as a power-saving measure that (unlike
  // background-tab throttling generally) applies to Web Audio API graphs
  // specifically - native <audio>/<video> elements are exempted while
  // audibly playing, which is why e.g. YouTube keeps playing when
  // minimized and this didn't. The suspension happens behind Howler's
  // back (its own internal state tracking never learns about it), so
  // nothing here self-heals without an explicit resume on return. Only
  // ever registered once, regardless of how many individual Howls end up
  // getting built.
  private registerContextResilienceListenersOnce() {
    if (this.contextListenersRegistered) return;
    this.contextListenersRegistered = true;
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden) Howler.ctx.resume();
    });
    window.addEventListener("focus", () => Howler.ctx.resume());
  }

  // Stops whatever's currently playing and plays the given index,
  // publishing the change to music-store so the UI (current-track
  // highlight, mini-player artwork) can react to it. Lazily builds the
  // target track's Howl right before playing it - the one choke point
  // every track change (onend/advance, next, previous, playTrackById)
  // already funnels through.
  private playIndex(index: number) {
    this.howls[this.currentIndex]?.stop();
    this.currentIndex = index;
    useMusicStore.getState().setCurrentIndex(index);
    if (this.started) this.ensureHowlBuilt(index)?.play();
  }

  // Every path that moves away from the current track to a new one
  // (natural end, Next, or clicking a track in the list) must go through
  // this so `previous()`'s back-stack stays consistent regardless of
  // which of those paths was used - previously only the onend/Next path
  // pushed here, so a click-to-play left history stale, which is what
  // made "Previous" unreliable.
  private recordHistory() {
    this.history.push(this.currentIndex);
    useMusicStore.getState().setCanGoBack(true);
  }

  // Moves forward one track, honoring the current shuffle setting - shared
  // by both a natural track-end (onend) and a manual "Next" press so the
  // two behave identically. Picks uniformly among every track except the
  // one that just finished when shuffling, so it never immediately repeats
  // a track back to back.
  private advance() {
    this.recordHistory();
    const shuffleEnabled = useMusicStore.getState().shuffleEnabled;
    let next = this.currentIndex;
    if (shuffleEnabled && this.tracks.length > 1) {
      while (next === this.currentIndex) {
        next = Math.floor(Math.random() * this.tracks.length);
      }
    } else if (this.tracks.length > 0) {
      next = (this.currentIndex + 1) % this.tracks.length;
    }
    this.playIndex(next);
  }

  start() {
    if (this.tracks.length === 0) {
      this.pendingStart = true;
      return;
    }
    if (this.started) return;
    this.started = true;
    this.ensureHowlBuilt(this.currentIndex)?.play();
    useMusicStore.getState().setCurrentIndex(this.currentIndex);
  }

  // Pauses the currently-active track in place (freezes its playback
  // position) rather than just silencing it - unlike the old mute-based
  // approach, a paused track genuinely stops the audio graph.
  pause() {
    this.howls[this.currentIndex]?.pause();
  }

  // Resumes the currently-active Howl from its paused position. Calling
  // .play() again on an already-loaded, paused Howl picks up where it
  // left off rather than restarting - this is what keeps pressing Play
  // from jumping to a new track (only onend/next/previous do that).
  resume() {
    this.howls[this.currentIndex]?.play();
  }

  // Manual "Next" - a user pressing skip counts as playback intent even if
  // nothing has started yet, so this can also cold-start the player.
  next() {
    this.started = true;
    useAudioStore.getState().setPlaying(true);
    this.advance();
  }

  // Steps back through play history (not just currentIndex - 1), so
  // "previous" is meaningful under shuffle too. No-ops if there's nothing
  // before the current track yet.
  previous() {
    const prevIndex = this.history.pop();
    if (prevIndex === undefined) return;
    useMusicStore.getState().setCanGoBack(this.history.length > 0);
    this.started = true;
    useAudioStore.getState().setPlaying(true);
    this.playIndex(prevIndex);
  }

  // Click-to-play from the expanded track list - jumps straight to a
  // specific track regardless of shuffle state.
  playTrackById(id: string) {
    const index = this.tracks.findIndex((t) => t.id === id);
    if (index === -1) return;
    this.recordHistory();
    this.started = true;
    useAudioStore.getState().setPlaying(true);
    this.playIndex(index);
  }

  // Current playback position in seconds, or seeks to `time` if given.
  // Scoped to whichever Howl is at currentIndex - returns 0 if that
  // track's audio hasn't loaded yet (Howler's own .seek() only returns a
  // number once loaded, otherwise returns the Howl itself for chaining).
  seek(time?: number): number {
    const howl = this.howls[this.currentIndex];
    if (!howl) return 0;
    if (time !== undefined) {
      howl.seek(time);
      return time;
    }
    const pos = howl.seek();
    return typeof pos === "number" ? pos : 0;
  }

  // Current track's duration in seconds, 0 until its Howl has loaded
  // (relevant for any track beyond the first, which stays lazily built
  // until actually navigated to).
  getDuration(): number {
    return this.howls[this.currentIndex]?.duration() ?? 0;
  }

  isStarted() {
    return this.started;
  }
}

export const howlerManager = new HowlerManager();
