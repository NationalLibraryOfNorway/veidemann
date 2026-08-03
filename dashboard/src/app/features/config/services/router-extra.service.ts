import { Injectable, inject } from '@angular/core';
import {NavigationEnd, Router} from '@angular/router';

@Injectable({
  providedIn: "root",
})

export class RouterExtraService {
  private router = inject(Router);


  private previousUrl: string = undefined;
  private currentUrl: string = undefined;

  constructor() {
    const router = this.router;


    this.currentUrl = this.router.url;
    router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {
        this.previousUrl = this.currentUrl;
        this.currentUrl = event.url;
      }
    });
  }

  public getPreviousUrl() {
    return this.previousUrl;
  }

  public getCurrentUrl() {
    return this.currentUrl;
  }
}
