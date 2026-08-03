import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {MatButtonModule} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-section-header',
  standalone: true,
  imports: [MatButtonModule, MatIcon, RouterLink],
  templateUrl: './section-header.component.html',
  styleUrls: ['./section-header.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SectionHeaderComponent {
  @Input({required: true}) rootLabel: string;
  @Input({required: true}) rootLink: string | readonly unknown[];
  @Input({required: true}) title: string;
  @Input({required: true}) backLink: string | readonly unknown[];
  @Input() listLink: string | readonly unknown[];
  @Input() detail = false;
}
