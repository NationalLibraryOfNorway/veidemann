import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {ConfigObject, Label} from '../../../../../shared/models';
import {MatNavList} from '@angular/material/list';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';

@Component({
  selector: 'app-entity-view',
  templateUrl: './entity-view.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatChipsModule,
    MatIcon,
    MatNavList,
    RouterLink
  ],
  standalone: true
})

export class EntityViewComponent {

  @Input()
  configObject: ConfigObject;

  constructor() {
  }

  get id(): string {
    return this.configObject.id;
  }

  get labels(): Label[] {
    return this.configObject.meta.labelList;
  }

  get name(): string {
    return this.configObject.meta.name;
  }

  get description(): string {
    return this.configObject.meta.description;
  }
}
