import {Component, EventEmitter, Input, Output} from '@angular/core';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {ActivatedRoute, Router} from '@angular/router';
import {ErrorService} from '../../../../core';
import {JobExecutionState} from '../../../../shared/models/report';
import {FilterShortcutComponent} from './filter-shortcut/filter-shortcut.component';
import {ActionShortcutComponent} from './action-shortcut/action-shortcut.component';
import {ShortcutListComponent} from './shortcut-list/shortcut-list.component';

@Component({
  selector: 'app-shortcut',
  templateUrl: './shortcut.component.html',
  styleUrls: ['./shortcut.component.scss'],
  imports: [
    ActionShortcutComponent,
    FilterShortcutComponent,
    ShortcutListComponent
  ],
  standalone: true
})
export class ShortcutComponent {
  readonly Kind = Kind;
  readonly JobExecutionState = JobExecutionState;
  @Input()
  configObject: ConfigObject;

  @Output()
  createSeed = new EventEmitter<ConfigObject>();

  @Output()
  runCrawl = new EventEmitter<ConfigObject>();

  @Output()
  clone = new EventEmitter<ConfigObject>();

  constructor(protected route: ActivatedRoute, protected router: Router, protected errorService: ErrorService) {
  }

  onClone() {
    this.clone.emit(this.configObject);
  }

  onCreateSeed() {
    this.createSeed.emit(this.configObject);
  }

  onRunCrawl() {
    this.runCrawl.emit(this.configObject);
  }

}
