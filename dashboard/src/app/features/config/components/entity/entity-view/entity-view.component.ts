import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {DatePipe} from '@angular/common';
import {ConfigObject, Label} from '../../../../../shared/models';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';
import {LabelDisplayComponent} from '../../../../../shared/components';

@Component({
  selector: 'app-entity-view',
  templateUrl: './entity-view.component.html',
  styleUrls: ['./entity-view.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatIcon,
    LabelDisplayComponent,
    RouterLink
  ],
  standalone: true
})

export class EntityViewComponent {

  @Input()
  configObject: ConfigObject;

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

  get created(): string {
    return this.configObject.meta.created;
  }

  get createdBy(): string {
    return this.configObject.meta.createdBy;
  }

  get lastModified(): string {
    return this.configObject.meta.lastModified;
  }

  get lastModifiedBy(): string {
    return this.configObject.meta.lastModifiedBy;
  }

  get hasFacts(): boolean {
    return !!(this.description || this.labels.length || this.created
      || this.createdBy || this.lastModified || this.lastModifiedBy);
  }
}
