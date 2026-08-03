import {Component, EventEmitter, Input, Output, ChangeDetectionStrategy} from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models/config';
import {AbilityServiceSignal} from "@casl/angular";
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
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: true
})
export class ActionShortcutComponent {
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<any>['can'];

  @Input()
  configObject: ConfigObject;

  @Output()
  createSeed = new EventEmitter();

  @Output()
  runCrawl = new EventEmitter();

  @Output()
  clone = new EventEmitter();


  constructor(private abilityService: AbilityServiceSignal<any>) {
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
