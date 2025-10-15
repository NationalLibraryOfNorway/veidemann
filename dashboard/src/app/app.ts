import {CommonModule} from "@angular/common";
import {Component} from "@angular/core";
import {initError} from "./app.config";
import {RouterOutlet} from '@angular/router';

@Component({
  standalone: true,
  selector: 'app-root',
  imports: [CommonModule, RouterOutlet],
  template: `
  @defer (when appReady()) {
    <router-outlet />
  } @error {
      <h1 i18n="@@appModuleInitFailedMessage">Initialization failed</h1>
      <p>{{ error.message }}</p>
  } @placeholder {
    <header>
      <h1 i18n="@@appLoadingText">Initializing...</h1>
    </header>
  } @loading {
    <header>
      <h1 i18n="@@appLoadingText">Loading...</h1>
    </header>
  }`
})
export class App {
  readonly error = initError();

  appReady = () => {
    return this.error ? Promise.reject(this.error) : Promise.resolve(true);
  };
}
