import { Injectable, OnDestroy, inject } from '@angular/core';
import { toObservable } from '@angular/core/rxjs-interop';
import { EMPTY, Subject, fromEvent, merge } from 'rxjs';
import { debounceTime, map, startWith, switchMap, takeUntil } from 'rxjs/operators';
import { AuthService } from './auth.service';

/** 10 minutes — PCI DSS-aligned inactivity threshold for fintech UIs. */
const IDLE_TIMEOUT_MS = 10 * 60 * 1000;

/**
 * IdleTimeoutService signs the user out after IDLE_TIMEOUT_MS of inactivity.
 *
 * Tracks user activity via: mousemove, keydown, click, scroll (passive),
 * touchstart (passive). Resets the timer on any tracked event.
 *
 * Monitoring starts when isAuthenticated becomes true and stops on sign-out.
 * The timer is implemented via RxJS debounceTime, which is throttled in
 * background tabs — the logout may fire later than intended but will not be skipped.
 */
@Injectable({ providedIn: 'root' })
export class IdleTimeoutService implements OnDestroy {
  private readonly auth = inject(AuthService);
  private readonly destroy$ = new Subject<void>();

  constructor() {
    // Bridge the isAuthenticated signal to an RxJS stream.
    // When authenticated, start the idle timer. On sign-out, stop and wait for re-auth.
    toObservable(this.auth.isAuthenticated)
      .pipe(
        switchMap((isAuth) => (isAuth ? this.idleTimer$() : EMPTY)),
        takeUntil(this.destroy$),
      )
      .subscribe(() => {
        void this.auth.signOut();
      });
  }

  /**
   * Returns an observable that emits once after IDLE_TIMEOUT_MS of inactivity.
   * Resets on any user activity event.
   */
  private idleTimer$() {
    const events$ = merge(
      fromEvent(document, 'mousemove'),
      fromEvent(document, 'keydown'),
      fromEvent(document, 'click'),
      fromEvent(document, 'scroll', { passive: true }),
      fromEvent(document, 'touchstart', { passive: true }),
    );

    return events$.pipe(
      startWith(null), // Emit immediately to start the debounce window.
      debounceTime(IDLE_TIMEOUT_MS),
      map(() => void 0),
    );
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}
