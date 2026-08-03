import { Component, EventEmitter, Input, Output, ChangeDetectionStrategy, inject } from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models/config';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-action-shortcut',
  templateUrl: './action-shortcut.component.html',
  imports: [
    MatIcon,
    MatListModule,
    MatTooltip
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class ActionShortcutComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input()
  configObject: ConfigObject;

  @Output()
  createSeed = new EventEmitter();

  @Output()
  runCrawl = new EventEmitter();

  @Output()
  clone = new EventEmitter();


  constructor() {
    this.can = this.abilityService.can;
  }

  onClone() {
    this.clone.emit();
  }

  onCreateSeed() {
    this.createSeed.emit();
  }

  onRunCrawl() {
    this.runCrawl.emit();
  }
}
