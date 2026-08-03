import {Component, Input, ChangeDetectionStrategy} from '@angular/core';
import {ConfigObject, RotationPolicy, SubCollectionType} from '../../../../../shared/models/config';
import {MatCheckbox} from '@angular/material/checkbox';
import {FileSizePipe} from '../../../../../shared/pipes/filesize.pipe';
import {MatLabel} from '@angular/material/form-field';
import {FormsModule} from '@angular/forms';

@Component({
  selector: 'app-collection-preview',
  templateUrl: './collection-preview.component.html',
  styleUrls: ['./collection-preview.component.css'],
  imports: [
    FormsModule,
    MatCheckbox,
    MatLabel,
    FileSizePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class CollectionPreviewComponent {
  readonly RotationPolicy = RotationPolicy;
  readonly SubCollectionType = SubCollectionType;

  @Input()
  configObject: ConfigObject;

}
