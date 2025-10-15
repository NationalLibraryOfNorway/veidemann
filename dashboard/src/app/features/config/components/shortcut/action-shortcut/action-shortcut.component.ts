import {Component, EventEmitter, Input, Output} from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models/config';
import {Observable} from "rxjs";
import {AbilityService} from "@casl/angular";
import {AsyncPipe} from '@angular/common';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-action-shortcut',
  templateUrl: './action-shortcut.component.html',
  imports: [
    AsyncPipe,
    MatIcon,
    MatListModule,
    MatTooltip
  ],
  standalone: true
})
export class ActionShortcutComponent {
  readonly Kind = Kind;
  readonly ability$: Observable<any>

  @Input()
  configObject: ConfigObject;

  @Output()
  createSeed = new EventEmitter();

  @Output()
  runCrawl = new EventEmitter();

  @Output()
  clone = new EventEmitter();


  constructor(private ableService: AbilityService<any>) {
    this.ability$ = this.ableService.ability$;
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

