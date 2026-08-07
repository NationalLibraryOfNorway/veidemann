import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {ActivatedRoute, RouterOutlet} from '@angular/router';

import {map, tap} from 'rxjs/operators';

import {
  BrowserScriptType,
  ConfigObject,
  RobotsPolicy,
  Role,
  RotationPolicy,
  SubCollectionType
} from '../../shared/models';
import {OptionsService} from './services';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';

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
    RouterOutlet
  ],
  standalone: true
})
export class ConfigComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly optionsService = inject(OptionsService);

  constructor() {
    this.route.data.pipe(
      map(data => data['options']),
      tap(options => this.optionsService.next(options)),
      takeUntilDestroyed(),
    ).subscribe();
  }
}
