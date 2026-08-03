import {ChangeDetectionStrategy, Component, OnInit, Signal} from '@angular/core';
import {ActivatedRoute, RouterOutlet} from '@angular/router';

import {Observable} from 'rxjs';

import {
  BrowserScriptType,
  ConfigObject,
  Kind,
  RobotsPolicy,
  Role,
  RotationPolicy,
  SubCollectionType
} from '../../shared/models';
import {OptionsService} from './services';
import {map, tap} from 'rxjs/operators';
import {configKindFromPath} from './func';
import {MatSidenavModule} from '@angular/material/sidenav';
import {ConfigNavListComponent} from './containers';
import {AsyncPipe} from '@angular/common';
import {toSignal} from '@angular/core/rxjs-interop';

export interface ConfigOptions {
  rotationPolicies?: RotationPolicy[];
  subCollectionTypes?: SubCollectionType[];
  crawlConfigs?: ConfigObject[];
  crawlScheduleConfigs?: ConfigObject[];
  browserConfigs?: ConfigObject[];
  collections?: ConfigObject[];
  politenessConfigs?: ConfigObject[];
  browserScripts?: ConfigObject[];
  browserScriptTypes?: BrowserScriptType[];
  robotsPolicies?: RobotsPolicy[];
  crawlJobs?: ConfigObject[];
  roles?: Role[];
  scopeScripts?: ConfigObject[];
}

@Component({
  templateUrl: './config.html',
  styleUrls: ['./config.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    ConfigNavListComponent,
    MatSidenavModule,
    RouterOutlet
  ],
  standalone: true
})
export class ConfigComponent implements OnInit {
  readonly Kind = Kind;

  options$: Observable<ConfigOptions>;
  readonly kind: Signal<Kind>;

  constructor(private route: ActivatedRoute,
              private optionsService: OptionsService) {
    this.kind = toSignal(
      this.route.paramMap.pipe(map(params => configKindFromPath(params.get('kind')))),
      {requireSync: true}
    );
  }

  ngOnInit(): void {
    this.options$ = this.route.data.pipe(
      map(data => data['options']),
      tap(options => this.optionsService.next(options))
    );
  }

}


