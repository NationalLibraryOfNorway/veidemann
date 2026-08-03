import { Injectable, inject } from '@angular/core';
import { HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import {Observable} from 'rxjs';
import {OAuthService} from 'angular-oauth2-oidc';
import {AppConfig} from '../../app.config';


@Injectable({
  providedIn: 'root'
})
export class TokenInterceptor implements HttpInterceptor {
  private oAuthService = inject(OAuthService);
  private appConfig = inject(AppConfig);


  intercept(request: HttpRequest<unknown>, next: HttpHandler): Observable<HttpEvent<unknown>> {

    const token = this.oAuthService.getIdToken();

    if (request.url.indexOf(this.appConfig.grpcWebUrl) !== -1 && token !== null) {
      const headers = request.headers.set('Authorization', 'Bearer ' + token);
      request = request.clone({headers});
    }

    return next.handle(request);
  }
}
