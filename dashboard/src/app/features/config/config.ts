import {ChangeDetectionStrategy, Component, OnInit} from '@angular/core';
import {ActivatedRoute, ParamMap, RouterOutlet} from '@angular/router';

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
import {KindService, OptionsService} from './services';
import {map, tap} from 'rxjs/operators';
import {ConfigPath} from './func';
import {MatDrawerContainer, MatSidenavModule} from '@angular/material/sidenav';
import {ConfigNavListComponent} from './containers';
import {AsyncPipe} from '@angular/common';

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
  kind$: Observable<Kind>;

  constructor(private route: ActivatedRoute,
              private kindService: KindService,
              private optionsService: OptionsService) {
  }

  ngOnInit(): void {
    // Subscribe to changes of the kind url parameter (see ConfigurationsRoutingModule).
    // Propagate change via kindService.
    this.kind$ = this.route.paramMap.pipe(
      map((params: ParamMap) => params.get('kind')),
      map((kind: string) => ConfigPath[kind]),
      tap(kind => this.kindService.next(kind))
    );

    this.options$ = this.route.data.pipe(
      map(data => data['options']),
      tap(options => this.optionsService.next(options))
    );
  }

}




